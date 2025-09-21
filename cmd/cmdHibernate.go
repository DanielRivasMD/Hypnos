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
	"errors"
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
	"github.com/ttacon/chalk"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

// TODO: add feature to specify only launching notification
// TODO: allow `duration` or `time`

////////////////////////////////////////////////////////////////////////////////////////////////////

var hibernateLauncherCmd = &cobra.Command{
	Use:     "hibernate",
	Short:   "Send a probe to hibernation",
	Long:    helpHibernate,
	Example: exampleHibernate,

	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeWorkflowNames,

	PreRun: preRunHibernate,
	Run:    runHibernate,
}

////////////////////////////////////////////////////////////////////////////////////////////////////

var hibernateWorkerCmd = &cobra.Command{
	Use:    "hibernate-run",
	Hidden: true,

	Run: hiddenRunHibernate,
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// TODO: review struct composition
var (
	launcher configPaths
	worker   configPaths
)

type configPaths struct {
	config     string
	probe      string
	script     string
	log        string
	group      string
	duration   time.Duration
	recurrent  bool
	iterations int
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func init() {
	rootCmd.AddCommand(hibernateLauncherCmd)
	rootCmd.AddCommand(hibernateWorkerCmd)

	hibernateLauncherCmd.Flags().StringVarP(&launcher.probe, "probe", "", "", "instance name (manual or default: <config>-<ts>)")
	hibernateLauncherCmd.Flags().StringVarP(&launcher.group, "group", "g", "", "group label for this probe")
	hibernateLauncherCmd.Flags().StringVarP(&launcher.log, "log", "", "", "log file basename (no .log)")
	hibernateLauncherCmd.Flags().StringVarP(&launcher.script, "script", "", "", "shell command to execute")
	hibernateLauncherCmd.Flags().DurationVarP(&launcher.duration, "duration", "", time.Hour, "how long to wait")
	hibernateLauncherCmd.Flags().BoolVarP(&launcher.recurrent, "recurrent", "", false, "repeat timer indefinitely")
	hibernateLauncherCmd.Flags().IntVarP(&launcher.iterations, "iterations", "", 0, "run this many times (0=unlimited if --recurrent)")

	hibernateWorkerCmd.Flags().StringVar(&worker.probe, "probe", "", "instance name")
	hibernateWorkerCmd.Flags().StringVar(&worker.group, "group", "", "group label for this probe")
	hibernateWorkerCmd.Flags().StringVar(&worker.log, "log", "", "log basename")
	hibernateWorkerCmd.Flags().StringVar(&worker.script, "script", "", "shell command to execute")
	hibernateWorkerCmd.Flags().DurationVar(&worker.duration, "duration", time.Hour, "how long to wait")
	hibernateWorkerCmd.Flags().BoolVar(&worker.recurrent, "recurrent", false, "")
	hibernateWorkerCmd.Flags().IntVar(&worker.iterations, "iterations", 0, "")
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func preRunHibernate(cmd *cobra.Command, args []string) {
	const op = "hypnos.hibernate.pre"

	if len(args) == 1 {
		// CONFIG MODE: pull everything from TOML
		if flags.verbose {
			fmt.Println("Running on Config mode...")
		}

		// declare workflow
		launcher.config = args[0]

		// discover matching workflow file
		files, err := domovoi.ReadDir(dirs.config, flags.verbose)
		horus.CheckErr(err, horus.WithOp(op), horus.WithCategory("env_error"), horus.WithMessage("reading config dir"))
		var foundV *viper.Viper
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".toml") {
				continue
			}
			path := filepath.Join(dirs.config, f.Name())
			v := viper.New()
			v.SetConfigFile(path)
			if err := v.ReadInConfig(); err != nil {
				continue
			}
			if v.IsSet("workflows." + launcher.config) {
				foundV = v
				break
			}
		}
		if foundV == nil {
			horus.CheckErr(
				errors.New(""),
				horus.WithMessage(fmt.Sprintf("workflow %s not found", launcher.config)),
				horus.WithFormatter(func(he *horus.Herror) string { return onelineErr(he.Message) }),
			)
		}

		// defaults
		if launcher.probe == "" {
			launcher.probe = launcher.config
			horus.CheckErr(cmd.Flags().Set("probe", launcher.probe), horus.WithOp(op), horus.WithMessage("setting default --probe"))
		}

		wf := foundV.Sub("workflows." + launcher.config)
		bindFlag(cmd, "script", wf)
		bindFlag(cmd, "probe", wf)
		bindFlag(cmd, "log", wf)

		// TODO: bind differnt types
		bindFlag(cmd, "duration", wf)
		bindFlag(cmd, "recurrent", wf)
		bindFlag(cmd, "iterations", wf)

		// log default
		if !cmd.Flags().Changed("log") {
			launcher.log = launcher.config
			horus.CheckErr(cmd.Flags().Set("log", launcher.log), horus.WithOp(op), horus.WithMessage("setting default --log"))
		}

	} else {

		// MANUAL MODE: require explicit flags
		if flags.verbose {
			fmt.Println("Running on Manual mode...")
		}

		horus.CheckEmpty(
			launcher.probe,
			"",
			horus.WithMessage("`--probe` is required"),
			horus.WithExitCode(2),
			horus.WithFormatter(func(he *horus.Herror) string { return chalk.Red.Color(he.Message) }),
		)
		horus.CheckEmpty(
			launcher.script,
			"",
			horus.WithMessage("`--script` is required"),
			horus.WithExitCode(2),
			horus.WithFormatter(func(he *horus.Herror) string { return chalk.Red.Color(he.Message) }),
		)
		horus.CheckEmpty(
			launcher.log,
			"",
			horus.WithMessage("`--log` is required"),
			horus.WithExitCode(2),
			horus.WithFormatter(func(he *horus.Herror) string { return chalk.Red.Color(he.Message) }),
		)
		horus.CheckEmpty(
			launcher.duration.String(),
			"",
			horus.WithMessage("`--duration` is required"),
			horus.WithExitCode(2),
			horus.WithFormatter(func(he *horus.Herror) string { return chalk.Red.Color(he.Message) }),
		)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func runHibernate(cmd *cobra.Command, args []string) {
	const op = "hypnos.hibernate.launch"

	meta := &probeMeta{
		Probe:      launcher.probe,
		Group:      launcher.group,
		Script:     launcher.script,
		LogPath:    filepath.Join(dirs.log, launcher.log+".log"),
		Duration:   launcher.duration,
		Recurrent:  launcher.recurrent,
		Iterations: launcher.iterations,
		Quiescence: time.Now(),
	}

	pid, err := spawnProbe(meta)
	horus.CheckErr(err, horus.WithOp(op), horus.WithMessage("spawning worker"))
	meta.PID = pid

	// Persist metadata
	metaFile := filepath.Join(dirs.probe, launcher.probe+".json")
	f, err := os.Create(metaFile)
	horus.CheckErr(err, horus.WithOp(op), horus.WithMessage("creating meta file"))
	defer f.Close()
	horus.CheckErr(
		json.NewEncoder(f).Encode(meta),
		horus.WithOp(op),
		horus.WithMessage("encoding metadata"),
	)

	fmt.Printf("%s: spawned downtime %s with PID %s\n",
		chalk.Green.Color("OK:"),
		chalk.Green.Color(launcher.probe),
		chalk.Green.Color(strconv.Itoa(pid)),
	)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func hiddenRunHibernate(cmd *cobra.Command, args []string) {
	const op = "hypnos.hibernate.work"

	// open log
	logFile := filepath.Join(dirs.log, worker.log+".log")
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	horus.CheckErr(err, horus.WithOp(op), horus.WithMessage("opening log file"))
	defer f.Close()

	log := func(format string, a ...any) {
		line := fmt.Sprintf(format, a...)
		fmt.Fprintln(os.Stderr, line)
		fmt.Fprintln(f, line)
	}

	log("Downtime %q started for %s", worker.probe, worker.duration)

	// TODO: iterations not working properly
	count := 0
	for {
		count++
		// schedule and wait
		done := make(chan struct{})
		runDowntime(worker.duration, func() {
			log("▸ timer fired, executing shell snippet")
			if err := domovoi.ExecSh(worker.script); err != nil {
				log("▸ command failed: %v", err)
			}
			log("▸ timer fired, sending notification")
			if err := notify("Hypnos-"+worker.probe, "Downtime complete"); err != nil {
				log("▸ notify failed: %v", err)
			} else {
				log("▸ notify succeeded")
			}
			close(done)
		})
		<-done

		// TODO: if iterations flags
		// if iterations specified, stop after that many
		if worker.iterations > 0 && count >= worker.iterations {
			break
		}
		// if not marked recurrent, run only once
		if !worker.recurrent {
			break
		}
		// otherwise loop again
		log("▸ iteration %d complete, restarting timer", count)
	}

	log("Downtime %q fully complete (ran %d times)", worker.probe, count)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// spawnProbe forks off a new "hibernate-run" worker process, piping its output into the log
func spawnProbe(meta *probeMeta) (int, error) {
	// TODO: add error handlers
	exe, _ := os.Executable()

	args := []string{
		"hibernate-run",
		"--probe", meta.Probe,
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
