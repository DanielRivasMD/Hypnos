# hypnos, scheduled silence with intent

[![License](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)

## Overview
Minimalist CLI for scheduling "downtime" timers that run scripts or send notifications
`hypnos` spawns background workers, keeps logs, tracks state, and lets you inspect or cancel timers—all under `~/.hypnos`


#### hibernate
Schedules a new downtime timer. You can either:

Pass all flags manually

```bash
hypnos hibernate \
  --duration 30m \
  --name focus-session \
  --log focus-log \
  --script "open -a Mail"
```
Or load defaults from a TOML workflow

```bash
hypnos hibernate --config deep-focus
```
On invocation, it:

Creates `~/.hypnos/{config,logs,meta,probe}`

Reads any `config/<name>.toml` for a `[workflows.<name>]` script

Spawns a probe process `(hibernate-run)` that sleeps, executes your script, notifies, then cleans up

Persists a JSON record under `~/.hypnos/meta`

#### scan
Lists all scheduled timers and their status. It:

Reads every *.json in `~/.hypnos/meta`

Probes each PID with `kill(pid, 0)` to mark it running or ended

Outputs a table of Name, PID, start time, duration (plus elapsed), and status

```bash
hypnos scan
```

#### stasis
Stops one or more active timers by name. For each:

Reads its metadata to find the PID

Sends `SIGTERM` to that process

Removes the PID file in `~/.hypnos/probes` and its JSON metadata

```bash
hypnos stasis focus-session deep-dive
```



# Technical Architecture

hypnos is a Go-based CLI that cleanly separates its launcher from its worker,  
persists per-instance state on disk, and schedules timers in-process.

## Core Framework

- Built with **Cobra** for command definitions and **Viper** for loading TOML workflows  
- On start, the launcher forks itself via `os.Executable()` + `exec.Command()`,  
  invoking a hidden `*-run` subcommand as the detached worker  

## Storage Layout (`~/.hypnos/`)

- **config/**  
  Stores `*.toml` workflow files. Each `[workflows.<name>]` table supplies defaults for the `--config` flag  

- **logs/**  
  Captures `stdout`/`stderr` from every background worker. Inspect these to see script output or debug failures  

- **meta/**  
  Holds one JSON file per invocation, with fields:  
  - `name` (instance name)  
  - `script` (shell snippet)  
  - `duration`  
  - `PID`  
  - `timestamp` (invocation time)  

- **probes/**  
  Tracks live PID files (`<name>.pid`) for each active worker. Use these to probe liveness or stop timers  

~/.hypnos/
├─ config/
├─ logs/
├─ meta/
└─ probes/

## Execution Model

- **`domovoi.ExecSh`** runs user-provided shell snippets under `/bin/sh -c`  
- **`runDowntime(duration, callback)`** schedules the timer in the worker process, blocks until expiration,  
  then invokes the callback to execute the script and send notifications  

Wraps errors and diagnostics with Horus for consistent reporting
With hypnos you get a transparent, inspectable, and scriptable way to “master your dreams” by scheduling precise, background notifications or actions.


```toml
# ~/.hypnos/config/tasks.toml

[workflows.mail]
# opens the Mail app when the timer fires
script = "open -a 'Mail'"

[workflows.backup]
# runs your backup-complete script
script = "/usr/local/bin/backup_complete.sh"

[workflows.focus]
# speaks a message at the end of a focus session
script = "say 'Focus session complete!'"
```


## Features

## Quickstart
```
```

## Installation

### **Language-Specific**
| Language   | Command                                                                 |
|------------|-------------------------------------------------------------------------|
| **Go**     | `go install github.com/DanielRivasMD/Hypnos@latest`                  |

### **Pre-built Binaries**
Download from [Releases](https://github.com/DanielRivasMD/Hypnos/releases).

## Usage

```
```

## Example
```
```

## Configuration

## Development

Build from source
```
git clone https://github.com/DanielRivasMD/Hypnos
cd Hypnos
```

## Language-Specific Setup

| Language | Dev Dependencies | Hot Reload           |
|----------|------------------|----------------------|
| Go       | `go >= 1.21`     | `air` (live reload)  |

## License
Copyright (c) 2025

See the [LICENSE](LICENSE) file for license details
