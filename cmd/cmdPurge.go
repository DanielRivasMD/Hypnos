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
	"path/filepath"
	"syscall"

	"github.com/DanielRivasMD/domovoi"
	"github.com/DanielRivasMD/horus"
	"github.com/spf13/cobra"
	"github.com/ttacon/chalk"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

var purgeCmd = &cobra.Command{
	Use:     "purge " + chalk.Dim.TextStyle(chalk.Italic.TextStyle("[probe]")),
	Short:   "Terminate and clean up probes",
	Long:    helpPurge,
	Example: examplePurge,

	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeProbeNames,

	Run: runPurge,
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func init() {
	rootCmd.AddCommand(purgeCmd)

	purgeCmd.Flags().BoolVar(&flags.purgeAll, "all", false, "Purge all probes")
	purgeCmd.Flags().StringVar(&flags.purgeGroup, "group", "", "Purge all probes in a specific group")

	// horus.CheckErr(
	// 	purgeCmd.RegisterFlagCompletionFunc("group", completeProbeGroups),
	// 	horus.WithOp("purge.init"),
	// 	horus.WithMessage("registering group completion"),
	// )
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func runPurge(cmd *cobra.Command, args []string) {
	const op = "hypnos.purge"

	switch {
	case flags.purgeAll:
		purgeAllProbes()
	// case flags.purgeGroup != "":
	// 	purgeGroupProbes(flags.purgeGroup)
	case len(args) == 1:
		purgeProbe(args[0])
	default:
		horus.CheckErr(
			errors.New(""),
			horus.WithOp(op),
			horus.WithMessage("probe / flag"),
			horus.WithExitCode(2),
			horus.WithFormatter(func(he *horus.Herror) string {
				return "missing " + onelineErr(he.Message)
			}),
		)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func purgeProbe(name string) {
	const op = "hypnos.purge"

	metaFile := filepath.Join(dirs.probe, name+".json")
	data, err := os.ReadFile(metaFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: metadata for %q not found (%v)\n", name, err)
		return
	}

	var m probeMeta
	if err := json.Unmarshal(data, &m); err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid metadata for %q (%v)\n", name, err)
		return
	}

	// try terminating process, but proceed if already gone
	if err := syscall.Kill(m.PID, syscall.SIGTERM); err != nil {
		if err == syscall.ESRCH {
			fmt.Printf("warning: process %d for %q not running\n", m.PID, name)
		} else {
			fmt.Fprintf(os.Stderr, "error: cannot kill PID %d for %q (%v)\n", m.PID, name, err)
			return
		}
	} else {
		fmt.Printf("%s sent SIGTERM to PID %d for %q\n", chalk.Green.Color("OK:"), m.PID, name)
	}

	// remove metadata JSON file
	horus.CheckErr(
		func() error {
			_, err := domovoi.RemoveFile(metaFile, flags.verbose)(metaFile)
			return err
		}(),
		horus.WithOp(op),
		horus.WithCategory("io_error"),
		horus.WithMessage("removing metadata file"),
	)

	// remove log file
	horus.CheckErr(
		func() error {
			_, err := domovoi.RemoveFile(m.LogPath, flags.verbose)(m.LogPath)
			return err
		}(),
		horus.WithOp(op),
		horus.WithCategory("io_error"),
		horus.WithMessage("removing log file"),
	)

	fmt.Printf("%s purged probe %q\n", chalk.Green.Color("OK:"), m.Probe)
}

// func purgeGroupProbes(group string) {
// 	for _, metaFile := range listProbeMetaFiles() {
// 		if matchProbeGroup(metaFile, group) {
// 			purgeProbe(stripProbeName(metaFile))
// 		}
// 	}
// }

func purgeAllProbes() {
	for _, metaFile := range listProbeMetaFiles() {
		purgeProbe(stripProbeName(metaFile))
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////
