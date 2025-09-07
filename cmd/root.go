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
	"path/filepath"
	"strconv"
	"time"

	"github.com/DanielRivasMD/domovoi"
	"github.com/DanielRivasMD/horus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ttacon/chalk"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

var rootCmd = &cobra.Command{
	Use:     "hypnos",
	Long:    helpRoot,
	Example: exampleRoot,
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func Execute() {
	horus.CheckErr(rootCmd.Execute())
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// TODO: add command to launch all jobs as recurrent, i.e., start session
var (
	verbose bool
)

var (
	home      string
	hypnosDir string
	configDir string
	logDir    string
	probeDir  string
)

////////////////////////////////////////////////////////////////////////////////////////////////////

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose diagnostics")
	cobra.OnInitialize(initConfigPaths)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// probeMeta holds persisted state for each probe invocation
type probeMeta struct {
	Name       string        `json:"name"`
	Script     string        `json:"script"`
	LogPath    string        `json:"log_path"`
	Duration   time.Duration `json:"duration"`
	Recurrent  bool          `json:"recurrent"`
	Iterations int           `json:"iterations"`
	PID        int           `json:"pid"`
	Quiescence time.Time     `json:"quiescence"`
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func initConfigPaths() {
	var err error
	home, err = domovoi.FindHome(verbose)
	horus.CheckErr(err, horus.WithCategory("init_error"), horus.WithMessage("getting home directory"))
	hypnosDir = filepath.Join(home, ".lilith")
	configDir = filepath.Join(hypnosDir, "config")
	logDir = filepath.Join(hypnosDir, "logs")
	probeDir = filepath.Join(hypnosDir, "probe")
}

func errorFmt(er string) string {
	return chalk.Bold.TextStyle(chalk.Red.Color(er))
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func bindFlag(cmd *cobra.Command, flagName string, cfg *viper.Viper) {
	const op = "cli.bindFlag"
	flags := cmd.Flags()

	// only bind if not manually set & config has key
	if flags.Changed(flagName) || !cfg.IsSet(flagName) {
		return
	}

	f := flags.Lookup(flagName)
	if f == nil {
		// no such flag
		return
	}

	// build string representation based on flag declared type
	var raw string
	switch f.Value.Type() {
	case "string":
		raw = cfg.GetString(flagName)
	case "int":
		raw = strconv.Itoa(cfg.GetInt(flagName))
	case "bool":
		raw = strconv.FormatBool(cfg.GetBool(flagName))
	default:
		// fallback: just use the string getter
		raw = cfg.GetString(flagName)
	}

	// set flag value
	if err := flags.Set(flagName, raw); err != nil {
		horus.CheckErr(
			horus.NewCategorizedHerror(
				op,
				"config_error",
				fmt.Sprintf("setting %q from config", flagName),
				err,
				map[string]any{
					"flag":  flagName,
					"value": raw,
				},
			),
		)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////
