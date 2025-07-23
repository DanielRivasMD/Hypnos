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
	"path/filepath"
	"syscall"

	"github.com/DanielRivasMD/domovoi"
	"github.com/DanielRivasMD/horus"
	"github.com/spf13/cobra"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

// stopCmd stops one or more running downtime timers by name.
var stopCmd = &cobra.Command{
	Use:   "stop [name ...]",
	Short: "Stop one or more downtime timers",
	Long: `Stops the named downtime instances. 
It will send SIGTERM to each worker process, remove its PID file and metadata.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		const op = "hypnos.stop"

		home, err := domovoi.FindHome(verbose)
		horus.CheckErr(err, horus.WithOp(op), horus.WithMessage("finding home"))

		base := filepath.Join(home, ".hypnos")
		metaDir := filepath.Join(base, "meta")
		pidDir := filepath.Join(base, "daemons")

		for _, name := range args {
			metaFile := filepath.Join(metaDir, name+".json")
			data, err := os.ReadFile(metaFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: metadata for %q not found (%v)\n", name, err)
				continue
			}

			// parse metadata to get PID
			var m struct{ PID int }
			if err := json.Unmarshal(data, &m); err != nil {
				fmt.Fprintf(os.Stderr, "error: invalid metadata for %q (%v)\n", name, err)
				continue
			}

			// attempt to kill
			if err := syscall.Kill(m.PID, syscall.SIGTERM); err != nil {
				if err == syscall.ESRCH {
					fmt.Printf("warning: process %d for %q not running\n", m.PID, name)
				} else {
					fmt.Fprintf(os.Stderr, "error: cannot kill PID %d for %q (%v)\n", m.PID, name, err)
					continue
				}
			} else {
				fmt.Printf("OK: sent SIGTERM to PID %d for %q\n", m.PID, name)
			}

			// remove pid file
			pidFile := filepath.Join(pidDir, name+".pid")
			if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "warning: cannot remove pid file %s (%v)\n", pidFile, err)
			}

			// remove metadata file
			if err := os.Remove(metaFile); err != nil {
				fmt.Fprintf(os.Stderr, "warning: cannot remove metadata %s (%v)\n", metaFile, err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}

////////////////////////////////////////////////////////////////////////////////////////////////////
