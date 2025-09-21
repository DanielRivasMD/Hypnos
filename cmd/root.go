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
	"strconv"
	"strings"
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

var (
	dirs  configDirs
	flags hypnosFlags
)

type configDirs struct {
	home   string
	hypnos string
	config string
	log    string
	probe  string
}

// TODO: add command to launch all jobs as recurrent, i.e., start session
type hypnosFlags struct {
	verbose bool

	configOutput string

	purgeAll   bool
	purgeGroup string
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func init() {
	rootCmd.PersistentFlags().BoolVarP(&flags.verbose, "verbose", "v", false, "Enable verbose diagnostics")
	cobra.OnInitialize(initConfigPaths)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// probeMeta holds persisted state for each probe invocation
type probeMeta struct {
	Probe      string        `json:"probe"`
	Group      string        `json:"group"`
	Script     string        `json:"script"`
	LogPath    string        `json:"log_path"`
	Duration   time.Duration `json:"duration"`
	Recurrent  bool          `json:"recurrent"`
	Iterations int           `json:"iterations"`
	PID        int           `json:"pid"`
	Quiescence time.Time     `json:"quiescence"`
	Notify     bool          `json:"notify"`
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func initConfigPaths() {
	var err error
	dirs.home, err = domovoi.FindHome(flags.verbose)
	horus.CheckErr(err, horus.WithCategory("init_error"), horus.WithMessage("getting home directory"))
	dirs.hypnos = filepath.Join(dirs.home, ".hypnos")
	dirs.config = filepath.Join(dirs.hypnos, "config")
	dirs.log = filepath.Join(dirs.hypnos, "log")
	dirs.probe = filepath.Join(dirs.hypnos, "probe")
}

func onelineErr(er string) string {
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

	var raw string
	switch f.Value.Type() {
	case "string":
		raw = cfg.GetString(flagName)

	case "int":
		raw = strconv.Itoa(cfg.GetInt(flagName))

	case "bool":
		raw = strconv.FormatBool(cfg.GetBool(flagName))

	case "duration":
		// viper stores durations as string, so parse and re-stringify
		val := cfg.GetString(flagName)
		if _, err := time.ParseDuration(val); err == nil {
			raw = val
		} else {
			horus.CheckErr(
				horus.NewCategorizedHerror(
					op,
					"config_error",
					fmt.Sprintf("invalid duration for %q", flagName),
					err,
					map[string]any{"value": val},
				),
			)
			return
		}

	case "float64":
		raw = strconv.FormatFloat(cfg.GetFloat64(flagName), 'f', -1, 64)

	default:
		// fallback: just use the string getter
		raw = cfg.GetString(flagName)
	}

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

// saveProbeMeta writes probe metadata to ~/.hypnos/probe/<name>.json
func saveProbeMeta(meta *probeMeta) {
	const op = "probe.saveMeta"

	// ensure the probe directory exists
	horus.CheckErr(
		domovoi.CreateDir(dirs.probe, flags.verbose),
		horus.WithOp(op),
		horus.WithCategory("io_error"),
		horus.WithMessage("creating probe directory"),
		horus.WithDetails(map[string]any{
			"dir": dirs.probe,
		}),
	)

	// marshal the metadata
	data, err := json.MarshalIndent(meta, "", "  ")
	horus.CheckErr(
		err,
		horus.WithOp(op),
		horus.WithCategory("encode_error"),
		horus.WithMessage("marshaling probe metadata"),
		horus.WithDetails(map[string]any{
			"probe": meta.Probe,
			"group": meta.Group,
		}),
	)

	// write the file
	path := filepath.Join(dirs.probe, meta.Probe+".json")
	horus.CheckErr(
		os.WriteFile(path, data, 0o644),
		horus.WithOp(op),
		horus.WithCategory("io_error"),
		horus.WithMessage("writing probe metadata file"),
		horus.WithDetails(map[string]any{
			"path": path,
		}),
	)
}

// loadProbeMeta reads ~/.hypnos/meta/<name>.json
func loadProbeMeta(name string) *probeMeta {
	const op = "hypnos.loadProbeMeta"

	path := filepath.Join(dirs.probe, name+".json")

	// read the file
	data, err := os.ReadFile(path)
	horus.CheckErr(
		err,
		horus.WithOp(op),
		horus.WithCategory("io_error"),
		horus.WithMessage("reading probe metadata file"),
		horus.WithDetails(map[string]any{
			"path": path,
			"name": name,
		}),
	)

	// unmarshal into struct
	var meta probeMeta
	horus.CheckErr(
		json.Unmarshal(data, &meta),
		horus.WithOp(op),
		horus.WithCategory("decode_error"),
		horus.WithMessage("unmarshaling probe metadata"),
		horus.WithDetails(map[string]any{
			"path": path,
			"name": name,
		}),
	)

	return &meta
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// listProbeMetaFiles returns the full paths of all probe metadata JSON files
func listProbeMetaFiles() []string {
	var files []string
	entries, err := os.ReadDir(dirs.probe)
	if err != nil {
		return files
	}
	for _, fi := range entries {
		if fi.IsDir() || !strings.HasSuffix(fi.Name(), ".json") {
			continue
		}
		files = append(files, filepath.Join(dirs.probe, fi.Name()))
	}
	return files
}

// stripProbeName takes a metadata file path and returns the bare probe name
// e.g. ~/.hypnos/meta/pmail.json -> "pmail"
func stripProbeName(metaFile string) string {
	base := filepath.Base(metaFile)
	return strings.TrimSuffix(base, ".json")
}

func matchProbeGroup(metaFile string, group string) bool {
	data, err := os.ReadFile(metaFile)
	if err != nil {
		return false
	}
	var m probeMeta
	if err := json.Unmarshal(data, &m); err != nil {
		return false
	}
	return m.Group == group
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// completeProbeNames suggests currently known probe names (from ~/.hypnos/meta/*.json)
func completeProbeNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var names []string

	files, err := os.ReadDir(dirs.probe)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	for _, fi := range files {
		if fi.IsDir() || !strings.HasSuffix(fi.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(fi.Name(), ".json")
		if strings.HasPrefix(name, toComplete) {
			names = append(names, name)
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

func completeProbeGroups(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	groups := make(map[string]struct{})
	for _, metaFile := range listProbeMetaFiles() {
		data, err := os.ReadFile(metaFile)
		if err != nil {
			continue
		}
		var m probeMeta
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}
		if m.Group != "" && strings.HasPrefix(m.Group, toComplete) {
			groups[m.Group] = struct{}{}
		}
	}
	var out []string
	for g := range groups {
		out = append(out, g)
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

// BUG: completion is detecting the names of the fields
func completeWorkflowNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	files, err := os.ReadDir(dirs.config)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	var opts []string
	for _, fi := range files {
		if fi.IsDir() || !strings.HasSuffix(fi.Name(), ".toml") {
			continue
		}
		path := filepath.Join(dirs.config, fi.Name())
		v := viper.New()
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			continue
		}
		for _, key := range v.AllKeys() {
			if wf, ok := strings.CutPrefix(key, "workflows."); ok {
				if strings.HasPrefix(wf, toComplete) {
					opts = append(opts, wf)
				}
			}
		}
	}
	return opts, cobra.ShellCompDirectiveNoFileComp
}

////////////////////////////////////////////////////////////////////////////////////////////////////
