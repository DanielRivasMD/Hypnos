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
	"strings"

	"github.com/DanielRivasMD/domovoi"
	"github.com/DanielRivasMD/horus"
	"github.com/spf13/cobra"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

var awakenCmd = &cobra.Command{
	Use:     "awaken",
	Short:   "",
	Long:    helpAwaken,
	Example: exampleAwaken,

	Run: runAwaken,
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func init() {
	rootCmd.AddCommand(awakenCmd)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func runAwaken(cmd *cobra.Command, args []string) {
	createSubdirs(dirs, flags.verbose)
	generateConfig(generateToml())
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func createSubdirs(d configDirs, verbose bool) {
	const op = "hypnos.awaken"

	toCreate := []struct {
		label, path string
	}{
		{"hypnos root", d.hypnos},
		{"config", d.config},
		{"log", d.log},
		{"probe", d.probe},
	}

	for _, dir := range toCreate {
		horus.CheckErr(
			domovoi.CreateDir(dir.path, verbose),
			horus.WithOp(op),
			horus.WithCategory("io_error"),
			horus.WithMessage(fmt.Sprintf("creating %s directory", dir.label)),
			horus.WithDetails(map[string]any{
				"path": dir.path,
			}),
		)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func generateToml() string {
	lines := []string{
		"# hypnos workflow configuration",
		"# Save this file as ~/.hypnos/config/<name>.toml",
		"# Each [workflows.<key>] defines a reusable timer preset",
		"",
		"[workflows.mail]",
		"# Shell command to execute when the timer expires",
		"script = \"open -a 'Mail'\"",
		"",
		"# Duration to wait before executing the script (supports 5s, 10m, 1h)",
		"duration = \"5s\"",
		"",
		"# Basename for the log file (saved under ~/.hypnos/logs/<log>.log)",
		"log = \"mail\"",
		"",
		"# Unique name for this probe instance (used for metadata and PID tracking)",
		"probe = \"pmail\"",
		"",
		"# Optional: repeat the timer indefinitely",
		"# recurrent = true",
		"",
		"# Optional: number of times to run the timer (ignored if recurrent = true)",
		"# iterations = 3",
	}

	return strings.Join(lines, "\n") + "\n"
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func generateConfig(example string) {
	op := "awaken.generateConfig"

	if flags.configOutput == "" {
		fmt.Print(example)
		return
	}

	horus.CheckErr(
		os.WriteFile(flags.configOutput, []byte(example), 0o644),
		horus.WithOp(op),
		horus.WithCategory("io_error"),
		horus.WithMessage(fmt.Sprintf("writing example to %q", flags.configOutput)),
	)

	fmt.Printf("Example config written to %s\n", flags.configOutput)
}

////////////////////////////////////////////////////////////////////////////////////////////////////
