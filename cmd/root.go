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
	"embed"
	"path/filepath"
	"sync"

	"github.com/DanielRivasMD/domovoi"
	"github.com/DanielRivasMD/horus"
	"github.com/spf13/cobra"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

//go:embed docs.json
var docsFS embed.FS

////////////////////////////////////////////////////////////////////////////////////////////////////

const (
	APP     = "hypnos"
	VERSION = "v0.1.0"
	AUTHOR  = "Daniel Rivas"
	EMAIL   = "danielrivasmd@gmail.com"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

var (
	onceRoot  sync.Once
	rootCmd   *cobra.Command
	rootFlags struct {
		verbose     bool
		stasisAll   bool
		stasisGroup string
	}
	configDirs configDir
)

type configDir struct {
	home   string
	hypnos string
	config string
	log    string
	probe  string
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func InitDocs() {
	info := domovoi.AppInfo{
		Name:    APP,
		Version: VERSION,
		Author:  AUTHOR,
		Email:   EMAIL,
	}
	domovoi.SetGlobalDocsConfig(docsFS, info)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func GetRootCmd() *cobra.Command {
	onceRoot.Do(func() {
		d := horus.Must(domovoi.GlobalDocs())
		var err error
		rootCmd, err = d.MakeCmd("root", nil)
		horus.CheckErr(err)

		rootCmd.PersistentFlags().BoolVarP(&rootFlags.verbose, "verbose", "v", false, "Enable verbose diagnostics")
		rootCmd.Version = VERSION

		cobra.OnInitialize(initConfigDirs)
	})
	return rootCmd
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func Execute() {
	horus.CheckErr(GetRootCmd().Execute())
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func initConfigDirs() {
	configDirs.home = func() string {
		h, e := domovoi.FindHome(rootFlags.verbose)
		horus.CheckErr(e, horus.WithCategory("init_error"), horus.WithMessage("getting home directory"))
		return h
	}()
	configDirs.hypnos = filepath.Join(configDirs.home, ".hypnos")
	configDirs.config = filepath.Join(configDirs.hypnos, "config")
	configDirs.log = filepath.Join(configDirs.hypnos, "log")
	configDirs.probe = filepath.Join(configDirs.hypnos, "probe")
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func BuildCommands() {
	root := GetRootCmd()
	root.AddCommand(
		CompletionCmd(),
		IdentityCmd(),

		HibernateLauncherCmd(),
		HibernateWorkerCmd(),
		PrimeCmd(),
		ScanCmd(),
		StasisCmd(),
	)
}

////////////////////////////////////////////////////////////////////////////////////////////////////
