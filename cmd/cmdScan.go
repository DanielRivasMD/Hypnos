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
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/DanielRivasMD/domovoi"
	"github.com/DanielRivasMD/horus"
	"github.com/spf13/cobra"
	"github.com/ttacon/chalk"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

var scanCmd = &cobra.Command{
	Use:     "scan",
	Short:   "List probes & their state",
	Long:    helpScan,
	Example: exampleScan,

	Run: runScan,
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func init() {
	rootCmd.AddCommand(scanCmd)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func runScan(cmd *cobra.Command, args []string) {
	const op = "hypnos.scan"

	// read the probe metadata directory
	entries, err := domovoi.ReadDir(dirs.probe, flags.verbose)
	horus.CheckErr(err, horus.WithOp(op), horus.WithMessage("reading probe directory"))

	if len(entries) == 0 {
		fmt.Println("no probes hibernating in ~/.hypnos/meta")
		return
	}

	// print header
	fmt.Printf(
		"%-20s %-15s %-6s %-20s %s\n",
		"NAME", "GROUP", "PID", "INVOKED", "STATUS",
	)

	// iterate over metadata files
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))

		// load metadata
		meta := loadProbeMeta(name)

		// determine process status via `ps`
		status := chalk.Red.Color("mortem")
		stateOut, err := exec.Command("ps", "-o", "state=", "-p", strconv.Itoa(meta.PID)).Output()
		if err == nil {
			state := strings.TrimSpace(string(stateOut))
			switch {
			case strings.HasPrefix(state, "T"):
				status = chalk.Yellow.Color("stasis")
			default:
				status = chalk.Green.Color("hibernating")
			}
		}

		// format invoked timestamp
		invoked := meta.Quiescence.Format("2006-01-02 15:04:05")

		// format duration (with age)
		age := time.Since(meta.Quiescence).Truncate(time.Second)
		duration := fmt.Sprintf("%s (%s ago)", meta.Duration, age)

		// print row
		fmt.Printf(
			"%-20s %-6d %-20s %-12s %s\n",
			meta.Probe, meta.PID, invoked, duration, status,
		)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////
