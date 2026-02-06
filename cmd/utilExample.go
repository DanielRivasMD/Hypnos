////////////////////////////////////////////////////////////////////////////////////////////////////

package cmd

////////////////////////////////////////////////////////////////////////////////////////////////////

import (
	"github.com/DanielRivasMD/domovoi"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

var exampleRoot = domovoi.FormatExample(
	"hypnos",
	[]string{
		"hypnos awaken",
		"hypnos hibernate --probe pmail --script \"open -a Mail\" --duration 5m",
		"hypnos scan",
		"hypnos stasis pmail",
	},
)

var exampleAwaken = domovoi.FormatExample(
	"hypnos awaken",
	[]string{
		"hypnos awaken",
		"hypnos awaken --config-output ~/.hypnos/config/example.toml",
	},
)

var exampleHibernate = domovoi.FormatExample(
	"hypnos hibernate",
	[]string{
		"hypnos hibernate --probe focus --script \"say 'Done'\" --duration 25m",
		"hypnos hibernate deep-focus",
		"hypnos hibernate --probe backup --script \"/usr/local/bin/backup.sh\" --duration 1h --recurrent",
	},
)

var exampleScan = domovoi.FormatExample(
	"hypnos scan",
	[]string{
		"hypnos scan",
		"hypnos scan --verbose",
	},
)

var exampleStasis = domovoi.FormatExample(
	"hypnos stasis",
	[]string{
		"hypnos stasis focus",
		"hypnos stasis --group deepwork",
		"hypnos stasis --all",
	},
)

////////////////////////////////////////////////////////////////////////////////////////////////////
