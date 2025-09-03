/*
Copyright © 2025 Daniel Rivas <danielrivasmd@gmail.com>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package cmd

////////////////////////////////////////////////////////////////////////////////////////////////////

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/DanielRivasMD/domovoi"
	"github.com/DanielRivasMD/horus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

// TODO: add feature to specify only launching notification
// TODO: allow `duration` or `time`

// probeMeta holds persisted state for each probe invocation
type probeMeta struct {
	Name       string        `json:"name"`
	Script     string        `json:"script"`
	LogPath    string        `json:"log_path"`
	Duration   time.Duration `json:"duration"`
	Recurrent  bool          `json:"recurrent"`
	Iterations int           `json:"iterations"`
	PID        int           `json:"pid"`
	Quiescence time.Time     `json:"quiescence"`
}

////////////////////////////////////////////////////////////////////////////////////////////////////

var (
	configName string

	// launcher flags
	launcherProbe      string
	launcherLog        string
	launcherScript     string
	launcherDuration   time.Duration
	launcherRecurrent  bool
	launcherIterations int

	// worker flags (hidden)
	workerProbe      string
	workerLog        string
	workerScript     string
	workerDuration   time.Duration
	workerRecurrent  bool
	workerIterations int
)

////////////////////////////////////////////////////////////////////////////////////////////////////

func init() {
	rootCmd.AddCommand(hibernateLauncherCmd)
	rootCmd.AddCommand(hibernateWorkerCmd)

	hibernateLauncherCmd.Flags().StringVarP(&launcherProbe, "probe", "", "", "instance name (manual or default: <config>-<ts>)")
	hibernateLauncherCmd.Flags().StringVarP(&launcherLog, "log", "", "", "log file basename (no .log)")
	hibernateLauncherCmd.Flags().StringVarP(&launcherScript, "script", "", "", "shell command to execute")
	hibernateLauncherCmd.Flags().DurationVarP(&launcherDuration, "duration", "", time.Hour, "how long to wait")
	hibernateLauncherCmd.Flags().BoolVarP(&launcherRecurrent, "recurrent", "", false, "repeat timer indefinitely")
	hibernateLauncherCmd.Flags().IntVarP(&launcherIterations, "iterations", "", 0, "run this many times (0=unlimited if --recurrent)")

	hibernateWorkerCmd.Flags().StringVar(&workerProbe, "probe", "", "instance name")
	hibernateWorkerCmd.Flags().StringVar(&workerLog, "log", "", "log basename")
	hibernateWorkerCmd.Flags().StringVar(&workerScript, "script", "", "shell command to execute")
	hibernateWorkerCmd.Flags().DurationVar(&workerDuration, "duration", time.Hour, "how long to wait")
	hibernateWorkerCmd.Flags().BoolVar(&workerRecurrent, "recurrent", false, "")
	hibernateWorkerCmd.Flags().IntVar(&workerIterations, "iterations", 0, "")
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// hibernateLauncherCmd is the user-facing command. It either accepts all flags manually:
//	hypnos hibernate --duration 5s --log in-vivo --name in-vivo --script 'open -a Program'
// or it loads defaults from a TOML:
//	hypnos hibernate probe
var hibernateLauncherCmd = &cobra.Command{
	Use:     "hibernate",
	Short:   "Send a probe to hibernation",
	Long:    helpHibernate,
	Example: exampleHibernate,

	PreRun: preRunHibernate,
	Run:    runHibernate,
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// hibernateWorkerCmd is the hidden worker
// sleeps, execs your command, sends notification, then exits
var hibernateWorkerCmd = &cobra.Command{
	Use:    "hibernate-run",
	Hidden: true,
	Run:    hiddenRunHibernate,
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func preRunHibernate(cmd *cobra.Command, args []string) {
	const op = "hypnos.hibernate.pre"
	configName = args[0]

	// Ensure our structure under ~/.hypnos
	home, err := domovoi.FindHome(verbose)
	if err != nil {
		fmt.Errorf("cannot find home: %w", err)
	}
	cfgDir := filepath.Join(home, ".hypnos", "config")

	for _, sub := range []string{"config", "logs", "meta", "probes"} {
		horus.CheckErr(
			domovoi.EnsureDirExist(filepath.Join(cfgDir, sub), verbose),
			horus.WithOp(op),
			horus.WithMessage("creating "+sub),
		)
	}

	var (
		foundV      *viper.Viper
		cfgFileUsed string
	)
	fis, err := domovoi.ReadDir(cfgDir, verbose)
	horus.CheckErr(
		err,
		horus.WithOp(op),
		horus.WithCategory("env_error"),
		horus.WithMessage("reading config dir"),
	)

	for _, fi := range fis {
		if fi.IsDir() || !strings.HasSuffix(fi.Name(), ".toml") {
			continue
		}
		path := filepath.Join(cfgDir, fi.Name())
		v := viper.New()
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			continue
		}
		if v.IsSet("workflows." + configName) {
			foundV = v
			cfgFileUsed = path
			break
		}
	}

	if foundV == nil {
		horus.CheckErr(
			fmt.Errorf("workflow %q not found in %s/*.toml", configName, cfgDir),
			horus.WithOp(op),
			horus.WithMessage("could not find named workflow in config directory"),
			horus.WithCategory("config_error"),
		)
	}

	if launcherProbe == "" {
		launcherProbe = configName
		horus.CheckErr(
			cmd.Flags().Set("probe", launcherProbe),
			horus.WithOp(op),
			horus.WithMessage("setting default --probe from config"),
			horus.WithCategory("config_error"),
		)
	}

	// TODO: patch variable
	fmt.Println(cfgFileUsed)
	// base := filepath.Base(cfgFileUsed)
	// groupName = strings.TrimSuffix(base, filepath.Ext(base))
	// horus.CheckErr(
	// 	cmd.Flags().Set("group", groupName),
	// 	horus.WithOp(op),
	// 	horus.WithMessage("setting default --group from TOML filename"),
	// 	horus.WithCategory("config_error"),
	// )

	wf := foundV.Sub("workflows." + configName)
	bindFlag(cmd, "script", &launcherScript, wf)
	bindFlag(cmd, "probe", &launcherProbe, wf)
	bindFlag(cmd, "log", &launcherLog, wf)

	// TODO: bind differnt types
	// bindFlag(cmd, "duration", &launcherDuration, wf)
	// bindFlag(cmd, "recurrent", &launcherRecurrent, wf)
	// bindFlag(cmd, "iterations", &launcherIterations, wf)

	// // helper to bind a flag only when unset
	// bind := func(name string, set func(string) error, fromV func() string) {
	// 	if !cmd.Flags().Changed(name) && wf.IsSet(name) {
	// 		_ = set(fromV())
	// 	}
	// }
	// bind("duration", cmd.Flags().Set, wf.GetString("duration"))
	// bind("recurrent", cmd.Flags().Set, strconv.FormatBool(wf.GetBool("recurrent")))
	// bind("iterations", cmd.Flags().Set, strconv.Itoa(wf.GetInt("iterations")))

	if !cmd.Flags().Changed("log") {
		launcherLog = configName
		horus.CheckErr(
			cmd.Flags().Set("log", launcherLog),
			horus.WithOp(op),
			horus.WithMessage("setting default --log from workflow key"),
			horus.WithCategory("config_error"),
		)
	}

}

// 	// If user passed --config, load it and extract defaults
// 	if cmd.Flags().Changed("config") {
// 		cfgDir := filepath.Join(cfgDir, "config")
// 		fis, err := domovoi.ReadDir(cfgDir, verbose)
// 		horus.CheckErr(err, horus.WithOp(op), horus.WithMessage("reading config dir"))

// 		var v *viper.Viper
// 		for _, fi := range fis {
// 			if fi.IsDir() || !strings.HasSuffix(fi.Name(), ".toml") {
// 				continue
// 			}
// 			path := filepath.Join(cfgDir, fi.Name())
// 			vv := viper.New()
// 			vv.SetConfigFile(path)
// 			if err := vv.ReadInConfig(); err != nil {
// 				continue
// 			}
// 			if vv.IsSet("workflows." + launcherConfig + ".script") {
// 				v = vv
// 				break
// 			}
// 		}
// 		if v == nil {
// 			fmt.Errorf("workflow %q not found in %s", launcherConfig, cfgDir)
// 		}

// 		wf := v.Sub("workflows." + launcherConfig)

// 		// Only set defaults if user did not override
// 		if !cmd.Flags().Changed("script") {
// 			launcherScript = wf.GetString("script")
// 			cmd.Flags().Set("script", launcherScript)
// 		}
// 	}
// }

////////////////////////////////////////////////////////////////////////////////////////////////////

func runHibernate(cmd *cobra.Command, args []string) {
	const op = "hypnos.hibernate.run"

	horus.CheckEmpty(
		launcherScript,
		"`--script` is required when --config is not provided",
		horus.WithOp(op),
		horus.WithMessage("provide a shell command to run"),
	)
	horus.CheckEmpty(
		launcherLog,
		"`--log` is required when --config is not provided",
		horus.WithOp(op),
		horus.WithMessage("provide a log basename"),
	)
	horus.CheckEmpty(
		launcherProbe,
		"`--name` is required when --config is not provided",
		horus.WithOp(op),
		horus.WithMessage("provide an instance name"),
	)

	home, err := domovoi.FindHome(verbose)
	horus.CheckErr(err, horus.WithOp(op), horus.WithMessage("finding home"))

	meta := &probeMeta{
		Name:       launcherProbe,
		Script:     launcherScript,
		LogPath:    filepath.Join(home, ".hypnos", "logs", launcherLog+".log"),
		Duration:   launcherDuration,
		Quiescence: time.Now(),
	}

	pid, err := spawnProbe(meta)
	horus.CheckErr(err, horus.WithOp(op), horus.WithMessage("spawning worker"))
	meta.PID = pid

	// Persist metadata
	metaFile := filepath.Join(home, ".hypnos", "meta", launcherProbe+".json")
	f, err := os.Create(metaFile)
	horus.CheckErr(err, horus.WithOp(op), horus.WithMessage("creating meta file"))
	defer f.Close()
	horus.CheckErr(
		json.NewEncoder(f).Encode(meta),
		horus.WithOp(op),
		horus.WithMessage("encoding metadata"),
	)

	fmt.Printf("OK: spawned downtime %q with PID %d\n", launcherProbe, pid)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func hiddenRunHibernate(cmd *cobra.Command, args []string) {
	const op = "hypnos.hibernate.run"

	home, err := domovoi.FindHome(verbose)
	horus.CheckErr(err, horus.WithOp(op), horus.WithMessage("finding home"))
	base := filepath.Join(home, ".hypnos")

	// pid file
	pidFile := filepath.Join(base, "probes", workerProbe+".pid")
	pid := os.Getpid()
	horus.CheckErr(
		os.WriteFile(pidFile,
			[]byte(fmt.Sprintf("%d\n", pid)), 0644),
		horus.WithOp(op),
		horus.WithMessage("writing pid file"),
	)
	defer os.Remove(pidFile)

	// open log
	logFile := filepath.Join(base, "logs", workerLog+".log")
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	horus.CheckErr(err, horus.WithOp(op), horus.WithMessage("opening log file"))
	defer f.Close()

	log := func(format string, a ...any) {
		line := fmt.Sprintf(format, a...)
		fmt.Fprintln(os.Stderr, line)
		fmt.Fprintln(f, line)
	}

	log("Downtime %q started for %s", workerProbe, workerDuration)

	count := 0
	for {
		count++
		done := make(chan struct{})
		runDowntime(workerDuration, func() {
			// your existing exec & notify code…
			close(done)
		})
		<-done

		// if iterations specified, stop after that many
		if workerIterations > 0 && count >= workerIterations {
			break
		}
		// if not marked recurrent, run only once
		if !workerRecurrent {
			break
		}
		// otherwise loop again
		log("▸ iteration %d complete, restarting timer", count)
	}

	log("Downtime %q fully complete (ran %d times)", workerProbe, count)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// spawnProbe forks off a new "hibernate-run" worker process, piping its output into the log
func spawnProbe(meta *probeMeta) (int, error) {
	exe, _ := os.Executable()
	args := []string{
		"hibernate-run",
		"--name", meta.Name,
		"--log", strings.TrimSuffix(filepath.Base(meta.LogPath), ".log"),
		"--script", meta.Script,
		"--duration", meta.Duration.String(),
	}
	if meta.Recurrent {
		args = append(args, "--recurrent")
	}
	if meta.Iterations > 0 {
		args = append(args, "--iterations", strconv.Itoa(meta.Iterations))
	}

	f, err := os.OpenFile(meta.LogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return 0, err
	}

	cmd := exec.Command(exe, args...)
	cmd.Stdout = f
	cmd.Stderr = f
	_ = cmd.Start()
	return cmd.Process.Pid, nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////
