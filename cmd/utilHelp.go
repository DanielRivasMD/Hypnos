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
	"Hypnos is a minimalist CLI for scheduling downtime timers that execute scripts or send\n"+
		"notifications after a delay. It manages background workers, persists metadata, keeps logs,\n"+
		"and provides commands to inspect, stop, or configure timers — all stored under ~/.hypnos.",
)

var helpAwaken = domovoi.FormatHelp(
	"Daniel Rivas",
	"danielrivasmd@gmail.com",
	"Initializes the Hypnos environment. Creates ~/.hypnos/{config,log,probe} and prints an\n"+
		"example workflow configuration. Use --config-output <file> to write the example to disk.",
)

var helpHibernate = domovoi.FormatHelp(
	"Daniel Rivas",
	"danielrivasmd@gmail.com",
	"Schedules a downtime timer. You may provide all flags manually, or pass a workflow name to\n"+
		"load defaults from ~/.hypnos/config/*.toml. The launcher spawns a hidden worker process\n"+
		"that sleeps for the specified duration, optionally executes a script, sends a notification,\n"+
		"and repeats based on --iterations or --recurrent. Metadata is saved under ~/.hypnos/probe.",
)

var helpScan = domovoi.FormatHelp(
	"Daniel Rivas",
	"danielrivasmd@gmail.com",
	"Lists all active or completed probes. Reads metadata from ~/.hypnos/probe/*.json and checks\n"+
		"each PID to determine whether the worker is running, stopped, or dead. Outputs a table with\n"+
		"probe name, group, PID, invocation time, duration, and status.",
)

var helpStasis = domovoi.FormatHelp(
	"Daniel Rivas",
	"danielrivasmd@gmail.com",
	"Stops one or more downtime probes. Sends SIGTERM to each worker process, then removes its\n"+
		"metadata and log files from ~/.hypnos/probe and ~/.hypnos/log. Supports purging a single\n"+
		"probe, all probes, or all probes in a specific group.",
)

////////////////////////////////////////////////////////////////////////////////////////////////////
