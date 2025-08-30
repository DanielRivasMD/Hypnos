////////////////////////////////////////////////////////////////////////////////////////////////////

package cmd

////////////////////////////////////////////////////////////////////////////////////////////////////

import (
	"github.com/ttacon/chalk"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

// formatHelp produces the “help” header + description.
//
//	author: name, e.g. "Daniel Rivas"
//	email:  email, e.g. "danielrivasmd@gmail.com"
//	desc:   the multi‐line description, "\n"-separated.
func formatHelp(author, email, desc string) string {
	header := chalk.Bold.TextStyle(
		chalk.Green.Color(author+" "),
	) +
		chalk.Dim.TextStyle(
			chalk.Italic.TextStyle("<"+email+">"),
		)

	// prefix two newlines to your desc, chalk it cyan + dim it
	body := "\n\n" + desc
	return header + chalk.Dim.TextStyle(chalk.Cyan.Color(body))
}

////////////////////////////////////////////////////////////////////////////////////////////////////

var helpRoot = formatHelp(
	"Daniel Rivas",
	"danielrivasmd@gmail.com",
	"",
)

var helpHibernate = formatHelp(
	"Daniel Rivas",
	"danielrivasmd@gmail.com",
	"Invoke a short-lived worker that waits, then runs your script\n"+
		"and pops a notification. You can pass all flags manually, or supply --config\n"+
		"to load a workflow from ~/.hypnos/config/*.toml.",
)

var helpScan = formatHelp(
	"Daniel Rivas",
	"danielrivasmd@gmail.com",
	"",
)

var helpStatis = formatHelp(
	"Daniel Rivas",
	"danielrivasmd@gmail.com",
	"Stops the named downtime instances\n"+
		"It will send SIGTERM to each worker process, remove its PID file and metadata",
)

////////////////////////////////////////////////////////////////////////////////////////////////////
