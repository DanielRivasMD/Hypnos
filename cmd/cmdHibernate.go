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
	"strings"
	"time"

	"github.com/DanielRivasMD/domovoi"
	"github.com/DanielRivasMD/horus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

// TODO: add feature to specify only launching notification
// TODO: add recurrent option
// TODO: allow `duration` or `time`

// probeMeta holds persisted state for each probe invocation
type probeMeta struct {
	Name       string        `json:"name"`
	Script     string        `json:"script"`
	LogPath    string        `json:"log_path"`
	Duration   time.Duration `json:"duration"`
	PID        int           `json:"pid"`
	Quiescence time.Time     `json:"quiescence"`
}

////////////////////////////////////////////////////////////////////////////////////////////////////

var (
	// launcher flags
	configName string
	probeName  string
	logName    string
	scriptPath string
	duration   time.Duration

	// worker flags (hidden)
	runName     string
	runLog      string
	runScript   string
	runDuration time.Duration
)

////////////////////////////////////////////////////////////////////////////////////////////////////

func init() {
	hibernateLauncherCmd.Flags().StringVarP(&configName, "config", "c", "", "load workflow from ~/.hypnos/config/<name>.toml")
	hibernateLauncherCmd.Flags().StringVarP(&probeName, "name", "n", "", "instance name (manual or default: <config>-<ts>)")
	hibernateLauncherCmd.Flags().StringVarP(&logName, "log", "l", "", "log file basename (no .log)")
	hibernateLauncherCmd.Flags().StringVarP(&scriptPath, "script", "s", "", "shell command to execute")
	hibernateLauncherCmd.Flags().DurationVarP(&duration, "duration", "t", time.Hour, "how long to wait")
	rootCmd.AddCommand(hibernateLauncherCmd)

	hibernateWorkerCmd.Flags().StringVar(&runName, "name", "", "instance name")
	hibernateWorkerCmd.Flags().StringVar(&runLog, "log", "", "log basename")
	hibernateWorkerCmd.Flags().StringVar(&runScript, "script", "", "shell command to execute")
	hibernateWorkerCmd.Flags().DurationVar(&runDuration, "duration", time.Hour, "how long to wait")
	rootCmd.AddCommand(hibernateWorkerCmd)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// hibernateLauncherCmd is the user-facing command. It either accepts all flags manually:
//
//	hypnos hibernate --duration 5s --log in-vivo --name in-vivo --script 'open -a Program'
//
// or it loads defaults from a TOML:
//
//	hypnos hibernate --config probe
var hibernateLauncherCmd = &cobra.Command{
	Use:     "hibernate",
	Short:   "Invoke a managed downtime timer",
	Long:    helpHibernate,
	Example: exampleHibernate,

	PreRun: preRunHibernate,

	Run: runHibernate,
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

	// Ensure our structure under ~/.hypnos
	home, err := domovoi.FindHome(verbose)
	if err != nil {
		fmt.Errorf("cannot find home: %w", err)
	}
	base := filepath.Join(home, ".hypnos")
	for _, sub := range []string{"config", "logs", "meta", "probes"} {
		horus.CheckErr(
			domovoi.EnsureDirExist(filepath.Join(base, sub), verbose),
			horus.WithOp(op),
			horus.WithMessage("creating "+sub),
		)
	}

	// If user passed --config, load it and extract defaults
	if cmd.Flags().Changed("config") {
		cfgDir := filepath.Join(base, "config")
		fis, err := domovoi.ReadDir(cfgDir, verbose)
		horus.CheckErr(err, horus.WithOp(op), horus.WithMessage("reading config dir"))

		var v *viper.Viper
		for _, fi := range fis {
			if fi.IsDir() || !strings.HasSuffix(fi.Name(), ".toml") {
				continue
			}
			path := filepath.Join(cfgDir, fi.Name())
			vv := viper.New()
			vv.SetConfigFile(path)
			if err := vv.ReadInConfig(); err != nil {
				continue
			}
			if vv.IsSet("workflows." + configName + ".script") {
				v = vv
				break
			}
		}
		if v == nil {
			fmt.Errorf("workflow %q not found in %s", configName, cfgDir)
		}

		wf := v.Sub("workflows." + configName)

		// Only set defaults if user did not override
		if !cmd.Flags().Changed("script") {
			scriptPath = wf.GetString("script")
			cmd.Flags().Set("script", scriptPath)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func runHibernate(cmd *cobra.Command, args []string) {
	const op = "hypnos.hibernate.run"

	// Manual flow: require all three flags if --config omitted
	if !cmd.Flags().Changed("config") {
		horus.CheckEmpty(
			scriptPath,
			"`--script` is required when --config is not provided",
			horus.WithOp(op),
			horus.WithMessage("provide a shell command to run"),
		)
		horus.CheckEmpty(
			logName,
			"`--log` is required when --config is not provided",
			horus.WithOp(op),
			horus.WithMessage("provide a log basename"),
		)
		horus.CheckEmpty(
			probeName,
			"`--name` is required when --config is not provided",
			horus.WithOp(op),
			horus.WithMessage("provide an instance name"),
		)
	}

	// Config flow: user can override --name / --log too,
	// otherwise we supply reasonable defaults
	if cmd.Flags().Changed("config") {
		if !cmd.Flags().Changed("name") {
			probeName = fmt.Sprintf("%s-%d", configName, time.Now().Unix())
		}
		if !cmd.Flags().Changed("log") {
			logName = configName
		}
	}

	// At this point both scriptPath, probeName & logName are guaranteed
	// Duration always comes from --duration
	horus.CheckEmpty(
		scriptPath,
		"`--script` is required",
		horus.WithOp(op),
		horus.WithMessage("no command to execute"),
	)

	home, err := domovoi.FindHome(verbose)
	horus.CheckErr(err, horus.WithOp(op), horus.WithMessage("finding home"))

	meta := &probeMeta{
		Name:       probeName,
		Script:     scriptPath,
		LogPath:    filepath.Join(home, ".hypnos", "logs", logName+".log"),
		Duration:   duration,
		Quiescence: time.Now(),
	}

	pid, err := spawnProbe(meta)
	horus.CheckErr(err, horus.WithOp(op), horus.WithMessage("spawning worker"))
	meta.PID = pid

	// Persist metadata
	metaFile := filepath.Join(home, ".hypnos", "meta", probeName+".json")
	f, err := os.Create(metaFile)
	horus.CheckErr(err, horus.WithOp(op), horus.WithMessage("creating meta file"))
	defer f.Close()
	horus.CheckErr(
		json.NewEncoder(f).Encode(meta),
		horus.WithOp(op),
		horus.WithMessage("encoding metadata"),
	)

	fmt.Printf("OK: spawned downtime %q with PID %d\n", probeName, pid)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func hiddenRunHibernate(cmd *cobra.Command, args []string) {
	const op = "hypnos.hibernate.run"

	home, err := domovoi.FindHome(verbose)
	horus.CheckErr(err, horus.WithOp(op), horus.WithMessage("finding home"))
	base := filepath.Join(home, ".hypnos")

	// pid file
	pidFile := filepath.Join(base, "probes", runName+".pid")
	pid := os.Getpid()
	horus.CheckErr(
		os.WriteFile(pidFile,
			[]byte(fmt.Sprintf("%d\n", pid)), 0644),
		horus.WithOp(op),
		horus.WithMessage("writing pid file"),
	)
	defer os.Remove(pidFile)

	// open log
	logFile := filepath.Join(base, "logs", runLog+".log")
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	horus.CheckErr(err, horus.WithOp(op), horus.WithMessage("opening log file"))
	defer f.Close()

	log := func(format string, a ...any) {
		line := fmt.Sprintf(format, a...)
		fmt.Fprintln(os.Stderr, line)
		fmt.Fprintln(f, line)
	}

	log("Downtime %q started for %s", runName, runDuration)

	done := make(chan struct{})
	runDowntime(runDuration, func() {
		log("▸ timer fired, executing shell snippet")
		if err := domovoi.ExecSh(runScript); err != nil {
			log("▸ command failed: %v", err)
		}
		if err := notify("Hypnos-"+runName, "Downtime complete"); err != nil {
			log("▸ notify failed: %v", err)
		} else {
			log("▸ notify succeeded")
		}
		close(done)
	})
	<-done

	log("Downtime %q complete", runName)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// spawnProbe forks off a new "hibernate-run" worker process, piping its output into the log
func spawnProbe(meta *probeMeta) (int, error) {
	exe, err := os.Executable()
	if err != nil {
		return 0, err
	}
	args := []string{
		"hibernate-run",
		"--name", meta.Name,
		"--log", strings.TrimSuffix(filepath.Base(meta.LogPath), ".log"),
		"--script", meta.Script,
		"--duration", meta.Duration.String(),
	}
	f, err := os.OpenFile(meta.LogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return 0, err
	}
	cmd := exec.Command(exe, args...)
	cmd.Stdout = f
	cmd.Stderr = f
	if err := cmd.Start(); err != nil {
		f.Close()
		return 0, err
	}
	return cmd.Process.Pid, nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////
