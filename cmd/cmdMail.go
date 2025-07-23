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
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

var (
	duration time.Duration
)

////////////////////////////////////////////////////////////////////////////////////////////////////

// mailCmd
var mailCmd = &cobra.Command{
	Use:   "mail",
	Short: "Start a downtime timer and notify when it ends",
	RunE: func(cmd *cobra.Command, args []string) error {

		////////////////////////////////////////////////////////////////////////////////////////////////////

		// Resolve local Hypnos data directories
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("unable to resolve home directory: %w", err)
		}
		baseDir := filepath.Join(home, ".hypnos")
		logDir := filepath.Join(baseDir, "logs")
		pidDir := filepath.Join(baseDir, "daemons")
		os.MkdirAll(logDir, 0755)
		os.MkdirAll(pidDir, 0755)

		////////////////////////////////////////////////////////////////////////////////////////////////////

		// Write PID file
		pid := os.Getpid()
		pidPath := filepath.Join(pidDir, fmt.Sprintf("mail-%d.pid", pid))
		if err := os.WriteFile(pidPath, []byte(fmt.Sprintf("%d\n", pid)), 0644); err != nil {
			return fmt.Errorf("unable to write pid file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "▸ [mail] running as pid=%d, recorded in %s\n", pid, pidPath)

		////////////////////////////////////////////////////////////////////////////////////////////////////

		// Open log file (optional)
		logPath := filepath.Join(logDir, fmt.Sprintf("mail-%d.log", pid))
		logFile, err := os.Create(logPath)
		if err != nil {
			return fmt.Errorf("unable to open log file: %w", err)
		}
		defer logFile.Close()

		// tee logs to stderr + file
		log := func(format string, args ...interface{}) {
			msg := fmt.Sprintf(format, args...)
			fmt.Fprintln(os.Stderr, msg)
			fmt.Fprintln(logFile, msg)
		}

		////////////////////////////////////////////////////////////////////////////////////////////////////

		// Setup channel for signaling
		done := make(chan struct{})
		log("Downtime started for %s", duration)

		runDowntime(duration, func() {
			log("▸ [mail] timer fired – attempting notification")
			err := notify("Hypnos", "Downtime complete")
			if err != nil {
				log("▸ [mail] notification failed: %v", err)
			} else {
				log("▸ [mail] notification succeeded")
			}
			close(done)
		})

		// Wait for completion
		<-done

		////////////////////////////////////////////////////////////////////////////////////////////////////

		// Clean up
		log("▸ [mail] downtime finished, removing pid file")
		os.Remove(pidPath)
		return nil
	},
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func init() {
	mailCmd.Flags().DurationVarP(&duration, "duration", "t", time.Hour, "how long to wait")
	rootCmd.AddCommand(mailCmd)
}

////////////////////////////////////////////////////////////////////////////////////////////////////
