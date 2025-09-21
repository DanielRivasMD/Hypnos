////////////////////////////////////////////////////////////////////////////////////////////////////

package cmd

////////////////////////////////////////////////////////////////////////////////////////////////////

import (
	"github.com/DanielRivasMD/domovoi"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

var helpRoot = domovoi.FormatHelp(
	"Daniel Rivas",
	"danielrivasmd@gmail.com",
	"",
)

var helpAwaken = domovoi.FormatHelp(
	"Daniel Rivas",
	"<danielrivasmd@gmail.com>",
	"",
)

var helpHibernate = domovoi.FormatHelp(
	"Daniel Rivas",
	"danielrivasmd@gmail.com",
	"Invoke a short-lived worker that waits, then runs your script\n"+
		"and pops a notification. You can pass all flags manually, or supply --config\n"+
		"to load a workflow from ~/.hypnos/config/*.toml.",
)

var helpScan = domovoi.FormatHelp(
	"Daniel Rivas",
	"danielrivasmd@gmail.com",
	"",
)

var helpPurge = domovoi.FormatHelp(
	"Daniel Rivas",
	"danielrivasmd@gmail.com",
	"Stops the named downtime instances\n"+
		"It will send SIGTERM to each worker process, remove its PID file and metadata",
)

////////////////////////////////////////////////////////////////////////////////////////////////////
