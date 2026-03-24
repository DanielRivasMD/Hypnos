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

type configPaths struct {
	config     string
	probe      string
	script     string
	log        string
	group      string
	duration   time.Duration
	recurrent  bool
	iterations int
	notify     bool
}

var (
	launcher configPaths
	worker   configPaths
)

////////////////////////////////////////////////////////////////////////////////////////////////////

func HibernateCmd() *cobra.Command {
	cmd := horus.Must(horus.Must(domovoi.GlobalDocs()).MakeCmd("hibernate", runHibernate,
		domovoi.WithArgs(cobra.MaximumNArgs(1)),
		domovoi.WithValidArgsFunction(completeWorkflowNames),
		domovoi.WithPreRun(preRunHibernate),
	))

	cmd.Flags().StringVarP(&launcher.probe, "probe", "", "", "instance name (manual or default: <config>-<ts>)")
	cmd.Flags().StringVarP(&launcher.group, "group", "g", "", "group label for this probe")
	cmd.Flags().StringVarP(&launcher.log, "log", "", "", "log file basename (no .log)")
	cmd.Flags().StringVarP(&launcher.script, "script", "", "", "shell command to execute")
	cmd.Flags().DurationVarP(&launcher.duration, "duration", "", time.Hour, "how long to wait")
	cmd.Flags().BoolVarP(&launcher.recurrent, "recurrent", "", false, "repeat timer indefinitely")
	cmd.Flags().IntVarP(&launcher.iterations, "iterations", "", 0, "run this many times (0=unlimited if --recurrent)")
	cmd.Flags().BoolVar(&launcher.notify, "notify-only", false, "only send notification, skip script execution")

	return cmd
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func HibernateWorkerCmd() *cobra.Command {
	cmd := horus.Must(horus.Must(domovoi.GlobalDocs()).MakeCmd("hibernate-run", hiddenRunHibernate))

	cmd.Flags().StringVar(&worker.probe, "probe", "", "instance name")
	cmd.Flags().StringVar(&worker.group, "group", "", "group label for this probe")
	cmd.Flags().StringVar(&worker.log, "log", "", "log basename")
	cmd.Flags().StringVar(&worker.script, "script", "", "shell command to execute")
	cmd.Flags().DurationVar(&worker.duration, "duration", time.Hour, "how long to wait")
	cmd.Flags().BoolVar(&worker.recurrent, "recurrent", false, "")
	cmd.Flags().IntVar(&worker.iterations, "iterations", 0, "")
	cmd.Flags().BoolVar(&worker.notify, "notify-only", false, "only send notification, skip script execution")

	return cmd
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func preRunHibernate(cmd *cobra.Command, args []string) {
	const op = "hypnos.hibernate.pre"

	if len(args) == 1 {
		// CONFIG MODE: pull everything from TOML
		if rootFlags.verbose {
			fmt.Println("Running on Config mode...")
		}

		launcher.config = args[0]

		files, err := domovoi.ReadDir(dirs.config, rootFlags.verbose)
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

		if launcher.probe == "" {
			launcher.probe = launcher.config
			horus.CheckErr(cmd.Flags().Set("probe", launcher.probe), horus.WithOp(op), horus.WithMessage("setting default --probe"))
		}

		wf := foundV.Sub("workflows." + launcher.config)
		bindFlag(cmd, "script", wf)
		bindFlag(cmd, "probe", wf)
		bindFlag(cmd, "log", wf)
		bindFlag(cmd, "duration", wf)
		bindFlag(cmd, "recurrent", wf)
		bindFlag(cmd, "iterations", wf)

		if !cmd.Flags().Changed("log") {
			launcher.log = launcher.config
			horus.CheckErr(cmd.Flags().Set("log", launcher.log), horus.WithOp(op), horus.WithMessage("setting default --log"))
		}
	} else {
		// MANUAL MODE: require explicit flags
		if rootFlags.verbose {
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
		Notify:     launcher.notify,
	}

	pid, err := spawnProbe(meta)
	horus.CheckErr(err, horus.WithOp(op), horus.WithMessage("spawning worker"))
	meta.PID = pid

	saveProbeMeta(meta)

	fmt.Printf("%s: spawned downtime %s with PID %s\n",
		chalk.Green.Color("OK:"),
		chalk.Green.Color(launcher.probe),
		chalk.Green.Color(strconv.Itoa(pid)),
	)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func hiddenRunHibernate(cmd *cobra.Command, args []string) {
	const op = "hypnos.hibernate.work"

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

		done := make(chan struct{})

		runDowntime(worker.duration, func() {
			if !worker.notify {
				log("▸ timer fired, executing shell snippet")
				if err := domovoi.ExecSh(worker.script); err != nil {
					log("▸ command failed: %v", err)
				}
			} else {
				log("▸ notify-only mode, skipping script execution")
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

		if worker.iterations > 0 && count >= worker.iterations {
			break
		}
		if worker.iterations == 0 && !worker.recurrent {
			break
		}

		log("▸ iteration %d complete, restarting timer", count)
	}

	log("Downtime %q fully complete (ran %d times)", worker.probe, count)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func spawnProbe(meta *probeMeta) (int, error) {
	exe, _ := os.Executable()

	args := []string{
		"hibernate-run",
		"--probe", meta.Probe,
		"--log", strings.TrimSuffix(filepath.Base(meta.LogPath), ".log"),
		"--script", meta.Script,
		"--duration", meta.Duration.String(),
	}

	if meta.Iterations > 0 {
		args = append(args, "--iterations", strconv.Itoa(meta.Iterations))
	} else if meta.Recurrent {
		args = append(args, "--recurrent")
	}
	if meta.Notify {
		args = append(args, "--notify-only")
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

func runDowntime(d time.Duration, onDone func()) {
	time.AfterFunc(d, onDone)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func notify(title, msg string) error {
	if tnPath, err := exec.LookPath("terminal-notifier"); err == nil {
		cmd := exec.Command(
			tnPath,
			"-title", title,
			"-message", msg,
			"-sender", "com.apple.Terminal",
		)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("terminal-notifier error: %v – %s", err, output)
		}
		return nil
	}

	if osaPath, err := exec.LookPath("osascript"); err == nil {
		script := fmt.Sprintf(`display notification %q with title %q`, msg, title)
		cmd := exec.Command(osaPath, "-e", script)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("osascript error: %v – %s", err, output)
		}
		return nil
	}

	return fmt.Errorf("no macOS notifier found: install terminal-notifier or ensure osascript is in PATH")
}

////////////////////////////////////////////////////////////////////////////////////////////////////
