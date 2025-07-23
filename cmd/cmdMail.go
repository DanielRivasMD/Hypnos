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
	"time"

	"github.com/spf13/cobra"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

// mailMeta holds persisted state for each downtime instance.
type mailMeta struct {
	Name      string        `json:"name"`
	Duration  time.Duration `json:"duration"`
	LogPath   string        `json:"log_path"`
	PID       int           `json:"pid"`
	InvokedAt time.Time     `json:"invoked_at"`
}

var (
	// instance name (defaults to timestamp)
	nameFlag string

	// log file basename (no “.log”)
	logName string

	// how long to wait
	duration time.Duration

	// internal: run in child mode
	childMode bool
)

////////////////////////////////////////////////////////////////////////////////////////////////////

var mailCmd = &cobra.Command{
	Use:   "mail",
	Short: "Invoke a managed downtime timer",
	Long: `Starts a named downtime instance, records its metadata in ~/.hypnos,
and spawns a background process to wait then notify.`,

	////////////////////////////////////////////////////////////////////////////////////////////////////

	PreRunE: func(cmd *cobra.Command, args []string) error {
		// 1) Ensure ~/.hypnos/{daemons,logs,meta} exist
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot find home dir: %w", err)
		}
		base := filepath.Join(home, ".hypnos")
		daemonDir := filepath.Join(base, "daemons")
		logDir := filepath.Join(base, "logs")
		metaDir := filepath.Join(base, "meta")
		if err := os.MkdirAll(daemonDir, 0755); err != nil {
			return err
		}
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return err
		}
		if err := os.MkdirAll(metaDir, 0755); err != nil {
			return err
		}

		// 2) Parent mode: spawn child, persist meta, then exit
		if !childMode {
			// default name & logName
			if nameFlag == "" {
				nameFlag = fmt.Sprintf("mail-%d", time.Now().Unix())
			}
			if logName == "" {
				logName = nameFlag
			}

			exe, err := os.Executable()
			if err != nil {
				return fmt.Errorf("locate executable: %w", err)
			}

			args := []string{
				"mail",
				"--child",
				"--name", nameFlag,
				"--duration", duration.String(),
				"--log", logName,
			}
			cmd := exec.Command(exe, args...)
			if err := cmd.Start(); err != nil {
				return fmt.Errorf("spawn child: %w", err)
			}

			// persist metadata
			meta := &mailMeta{
				Name:      nameFlag,
				Duration:  duration,
				LogPath:   filepath.Join(logDir, logName+".log"),
				PID:       cmd.Process.Pid,
				InvokedAt: time.Now(),
			}
			metaFile := filepath.Join(metaDir, nameFlag+".json")
			f, err := os.Create(metaFile)
			if err != nil {
				return fmt.Errorf("write meta: %w", err)
			}
			defer f.Close()
			if err := json.NewEncoder(f).Encode(meta); err != nil {
				return fmt.Errorf("encode meta: %w", err)
			}

			fmt.Printf("OK: spawned downtime %q with PID %d\n", nameFlag, meta.PID)
			os.Exit(0)
		}

		// 3) Child mode simply returns to Run()
		return nil
	},

	////////////////////////////////////////////////////////////////////////////////////////////////////

	Run: func(cmd *cobra.Command, args []string) {
		// 4) Child mode: write pidfile, open log, run & notify
		home, _ := os.UserHomeDir()
		base := filepath.Join(home, ".hypnos")
		pidPath := filepath.Join(base, "daemons", nameFlag+".pid")
		logPath := filepath.Join(base, "logs", logName+".log")

		// write pidfile
		pid := os.Getpid()
		os.WriteFile(pidPath, []byte(fmt.Sprintf("%d\n", pid)), 0644)

		// open log
		logF, err := os.Create(logPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "log error: %v\n", err)
			return
		}
		defer logF.Close()
		log := func(format string, args ...interface{}) {
			line := fmt.Sprintf(format, args...)
			fmt.Fprintln(os.Stderr, line)
			fmt.Fprintln(logF, line)
		}

		log("Downtime %q started for %s", nameFlag, duration)

		// schedule and wait
		done := make(chan struct{})
		runDowntime(duration, func() {
			log("▸ timer fired, sending notification")
			if err := notify("Hypnos-"+nameFlag, "Downtime complete"); err != nil {
				log("▸ notify failed: %v", err)
			} else {
				log("▸ notify succeeded")
			}
			close(done)
		})
		<-done

		// cleanup
		log("Downtime %q complete, removing pid file", nameFlag)
		os.Remove(pidPath)
	},
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func init() {
	mailCmd.Flags().StringVarP(&nameFlag, "name", "n", "", "instance name (default timestamp)")
	mailCmd.Flags().StringVarP(&logName, "log", "l", "", "log file basename")
	mailCmd.Flags().DurationVarP(&duration, "duration", "t", time.Hour, "how long to wait")
	mailCmd.Flags().BoolVar(&childMode, "child", false, "internal: child process mode")

	rootCmd.AddCommand(mailCmd)
}

////////////////////////////////////////////////////////////////////////////////////////////////////
