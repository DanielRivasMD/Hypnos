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

import (
	"fmt"
	"os"
	"time"

	daemon "github.com/sevlyar/go-daemon"
	"github.com/spf13/cobra"
)

var (
	// flag to run the command as a background daemon
	daemonize bool

	// how long the “downtime” should last
	duration time.Duration
)

////////////////////////////////////////////////////////////////////////////////////////////////////

var mailCmd = &cobra.Command{
	Use:   "mail",
	Short: "Start a downtime timer and notify when it ends",
	RunE: func(cmd *cobra.Command, args []string) error {

		// 1) Debug: print whether we're daemonizing
		fmt.Fprintf(os.Stderr, "⟳ debug: daemonize=%v\n", daemonize)

		// Setup the daemon context
		cntxt := &daemon.Context{
			PidFileName: "hypnos.pid",
			PidFilePerm: 0644,
			LogFileName: "hypnos.log",
			LogFilePerm: 0640,
			WorkDir:     "./",
			Umask:       027,
		}

		// 2) If requested, fork into background
		if daemonize {
			cwd, _ := os.Getwd()
			fmt.Fprintf(os.Stderr, "▸ [parent] forking from cwd %q…\n", cwd)
			child, err := cntxt.Reborn()
			if err != nil {
				return fmt.Errorf("unable to daemonize: %w", err)
			}
			if child != nil {
				// parent process: log child PID and exit
				fmt.Fprintf(os.Stderr, "▸ [parent] spawned child pid=%d\n", child.Pid)
				return nil
			}
			// child process continues below
			defer cntxt.Release()
			pid := os.Getpid()
			fmt.Fprintf(os.Stderr,
				"▸ [daemon] running as pid=%d, writing pidfile %s\n",
				pid, cntxt.PidFileName,
			)
		}

		////////////////////////////////////////////////////////////////////////////////////////////////
		// Kick off the downtime timer with notification callback
		//
		// We use a channel to block until the callback runs,
		// then exit cleanly (no deadlock).
		done := make(chan struct{})

		fmt.Printf("Downtime started for %s (daemon=%v)\n", duration, daemonize)
		RunDowntime(duration, func() {
			if err := Notify("Hypnos", "Downtime complete"); err != nil {
				fmt.Fprintf(os.Stderr, "notify failed: %v\n", err)
			}
			// signal that callback has finished
			close(done)
		})

		////////////////////////////////////////////////////////////////////////////////////////////////
		// If we're daemonized, wait for the callback; else return immediately
		if daemonize {
			<-done
		}
		return nil
	},
}

func init() {
	// attach flags
	mailCmd.Flags().BoolVarP(&daemonize, "daemon", "d", false, "run in background")
	mailCmd.Flags().DurationVarP(&duration, "duration", "t", time.Hour, "how long to wait")
	rootCmd.AddCommand(mailCmd)
}

////////////////////////////////////////////////////////////////////////////////////////////////////
