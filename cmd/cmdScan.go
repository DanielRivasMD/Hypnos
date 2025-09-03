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
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/DanielRivasMD/domovoi"
	"github.com/DanielRivasMD/horus"
	"github.com/spf13/cobra"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

// probeEntry mirrors the JSON you wrote in hibernateCmd
type probeEntry struct {
	Name       string        `json:"name"`
	LogPath    string        `json:"log_path"`
	Duration   time.Duration `json:"duration"`
	PID        int           `json:"pid"`
	Quiescence time.Time     `json:"quiescence"`
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func init() {
	rootCmd.AddCommand(scanCmd)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

var scanCmd = &cobra.Command{
	Use:     "scan",
	Short:   "List probes & their state",
	Long:    helpScan,
	Example: exampleScan,

	Run: runScan,
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func runScan(cmd *cobra.Command, args []string) {
	const op = "hypnos.scan"

	// TODO: double check location of active probes
	// 1) find ~/.hypnos/meta
	home, err := domovoi.FindHome(verbose)
	horus.CheckErr(err, horus.WithOp(op), horus.WithMessage("finding home dir"))

	metaDir := filepath.Join(home, ".hypnos", "meta")
	fis, err := os.ReadDir(metaDir)
	if err != nil {
		horus.CheckErr(err, horus.WithOp(op), horus.WithMessage("reading meta directory"))
	}
	if len(fis) == 0 {
		fmt.Println("no probes hibernating in ~/.hypnos/meta")
		return
	}

	// 2) load each .json, check pid
	entries := make([]probeEntry, 0, len(fis))
	for _, fi := range fis {
		if fi.IsDir() || !strings.HasSuffix(fi.Name(), ".json") {
			continue
		}
		path := filepath.Join(metaDir, fi.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: cannot read %s: %v\n", fi.Name(), err)
			continue
		}
		var m probeEntry
		if err := json.Unmarshal(data, &m); err != nil {
			fmt.Fprintf(os.Stderr, "warning: invalid JSON in %s: %v\n", fi.Name(), err)
			continue
		}
		entries = append(entries, m)
	}

	// 3) print table
	w := tabwriter.NewWriter(os.Stdout, 4, 8, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tPID\tINVOKED\tDURATION\tSTATUS")
	now := time.Now()
	for _, m := range entries {
		// kill(pid, 0) to detect existence
		running := true
		if err := syscall.Kill(m.PID, 0); err != nil {
			running = false
		}
		status := "ended"
		if running {
			status = "running"
		}
		// elapsed since invoked
		age := now.Sub(m.Quiescence).Truncate(time.Second)
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\n",
			m.Name,
			m.PID,
			m.Quiescence.Format("2006-01-02 15:04:05"),
			fmt.Sprintf("%s (%s ago)", m.Duration, age),
			status,
		)
	}
	w.Flush()
}

////////////////////////////////////////////////////////////////////////////////////////////////////
