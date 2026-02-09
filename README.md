# hypnos, scheduled silence with intent

[![License](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)

## Overview
Minimalist CLI for scheduling "downtime" timers that run scripts or send notifications.

`hypnos` spawns background workers, keeps logs, tracks state, and lets inspecting or canceling timers — all under `~/.hypnos`.

Lifecycle:

```
┌───────────────┐
│ hypnos awaken │  → creates directories + example config
└───────┬───────┘
        │
        ▼
┌─────────────────────────────┐
│ hypnos hibernate <workflow> │ → launcher loads config or flags
└───────┬─────────────────────┘
        │
        ▼
┌──────────────────────────────────────────┐
│ ┌──────────────┐                         │
│ │ spawnProbe() │ → starts worker process │
│ └──────┬───────┘                         │ 
│        │                                 │
│        ▼                                 │
│ ┌───────────────────────────────┐        │
│ │ hypnos hibernate-run (worker) │        │
│ │ - sleeps                      │        │
│ │ - executes script             │        │
│ │ - sends notifications         │        │
│ │ - logs output                 │        │
│ │ - repeats if needed           │        │
│ └──────┬────────────────────────┘        │
│        │                                 │
│        ▼                                 │
│ ┌────────────────┐                       │
│ │ probeMeta.json │ → updated metadata    │
│ └────────────────┘                       │
└───────┬──────────────────────────────────┘
        │
        ▼
┌─────────────┐
│ hypnos scan │ → monitors status
└───────┬─────┘
        │
        ▼
┌───────────────┐
│ hypnos stasis │ → kills process + removes files
└───────────────┘
```

#### awaken

Bootstraps the Hypnos environment.

    hypnos awaken

Creates the full directory layout:

    ~/.hypnos/
    ├─ config/   # workflow definitions (*.toml)
    ├─ log/      # logs for each probe (*.log)
    └─ probe/    # metadata for each running probe (*.json)

Prints an example TOML workflow, or writes it to a file if --config-output is set.


#### hibernate

Schedules a new downtime timer.

Manual mode:

    hypnos hibernate \
      --probe focus \
      --script "open -a Mail" \
      --log focus \
      --duration 30m

Workflow mode:

    hypnos hibernate deep-focus

Where ~/.hypnos/config/focus-session.toml contains:

    [workflows.focus-session]
    script = "open -a 'Mail'"
    duration = "30m"
    log = "focus"
    probe = "focus"

What hibernate does:

- Loads workflow defaults (if a name is provided)
- Validates required fields
- Builds a probeMeta record
- Spawns a background worker:

      hypnos hibernate-run --probe ... --duration ...

- Saves metadata to:

      ~/.hypnos/probe/<probe>.json

- Logs output to:

      ~/.hypnos/log/<log>.log


#### scan

Lists all active or completed probes.

    hypnos scan

- Reads all *.json metadata files under ~/.hypnos/probe
- Checks each PID using ps
- Prints:

```
|---------------------------------------|
| NAME | GROUP | PID | INVOKED | STATUS |
|---------------------------------------|
```

Statuses:

- hibernating → process running
- stasis      → process stopped (T state)
- mortem      → process no longer exists


#### stasis

Terminates and cleans up probes.

Stop a single probe:

    hypnos stasis focus

Stop all probes:

    hypnos stasis --all

Stop all probes in a group:

    hypnos stasis --group deepwork

What stasis does:

- Loads metadata (probeMeta)
- Sends SIGTERM to the worker PID
- Removes:

      ~/.hypnos/probe/<probe>.json
      ~/.hypnos/log/<log>.log


# Technical Architecture

Hypnos is a Go-based CLI that cleanly separates its launcher from its worker,
persists per-instance state on disk, and schedules timers in-process.

## Core Framework

- Built with Cobra for command definitions and Viper for loading TOML workflows
- The launcher forks itself via os.Executable() + exec.Command(),
  invoking a hidden "hibernate-run" subcommand as the detached worker

## Storage Layout (~/.hypnos/)

    ~/.hypnos/
    ├─ config/   # workflow definitions (*.toml)
    ├─ log/      # logs for each probe (*.log)
    └─ probe/    # metadata for each running probe (*.json)

Metadata fields (probeMeta):

- probe
- group
- script
- log_path
- duration
- recurrent
- iterations
- pid
- quiescence
- notify


## Workflow Configuration Example

    # ~/.hypnos/config/tasks.toml

    [workflows.mail]
    script = "open -a 'Mail'"
    duration = "5s"
    log = "mail"
    probe = "pmail"

    [workflows.backup]
    script = "/usr/local/bin/backup_complete.sh"
    duration = "1h"
    log = "backup"
    probe = "pbackup"

    [workflows.focus]
    script = "say 'Focus session complete!'"
    duration = "25m"
    log = "focus"
    probe = "pfocus"


## Features

(coming soon)


## Quickstart

(coming soon)


## Installation

### Language-Specific

    Go:  go install github.com/DanielRivasMD/Hypnos@latest

### Pre-built Binaries

Download from Releases.


## Usage

(coming soon)


## Example

(coming soon)


## Configuration

(coming soon)


## Development

Build from source:

    git clone https://github.com/DanielRivasMD/Hypnos
    cd Hypnos


## Language-Specific Setup

| Language | Dev Dependencies | Hot Reload |
|----------|------------------|------------|
| Go       | go >= 1.21       | air        |


## License

Copyright (c) 2025  
See the LICENSE file for license details.
