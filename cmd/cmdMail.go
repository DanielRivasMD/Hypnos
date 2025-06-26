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

var mailCmd = &cobra.Command{
	Use:   "mail",
	Short: "Start a downtime timer and notify when it ends",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Setup daemon context
		cntxt := &daemon.Context{
			PidFileName: "hypnos.pid",
			PidFilePerm: 0644,
			LogFileName: "hypnos.log",
			LogFilePerm: 0640,
			WorkDir:     "./",
			Umask:       027,
		}

		// If requested, fork to background
		if daemonize {
			child, err := cntxt.Reborn()
			if err != nil {
				return fmt.Errorf("unable to daemonize: %w", err)
			}
			if child != nil {
				// parent exits immediately
				return nil
			}
			// child continues:
			defer cntxt.Release()
		}

		// Kick off the timer
		fmt.Printf("Downtime started for %s (daemon=%v)\n", duration, daemonize)
		RunDowntime(duration, func() {
			if err := Notify("Hypnos", "Downtime complete"); err != nil {
				fmt.Fprintf(os.Stderr, "notify failed: %v\n", err)
			}
		})

		// If we're daemonized, block forever; otherwise exit immediately
		if daemonize {
			select {}
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
