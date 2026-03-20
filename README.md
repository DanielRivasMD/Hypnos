# hypnos, scheduled dreams with intent

[![License](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)

## Overview

CLI for scheduling "downtime" timers that run scripts or send notifications

`hypnos` spawns background sleepers, keeping logs, tracking state, and letting
inspecting or canceling timers

### Core Framework

- Built with **Cobra** for command definitions & **Viper** for loading TOML
  workflows
- The launcher forks itself via os.Executable() + exec.Command(), invoking a
  hidden "hibernate-private" subcommand as the detached sleeper

### Logic Schematic

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
    ┌───────────────────────────────────────────┐
    │ ┌──────────────┐                          │
    │ │ spawnProbe() │ → starts sleeper process │
    │ └──────┬───────┘                          │
    │        │                                  │
    │        ▼                                  │
    │ ┌────────────────────────────────┐        │
    │ │ hypnos hibernate-run (sleeper) │        │
    │ │ - sleeps                       │        │
    │ │ - executes script              │        │
    │ │ - sends notifications          │        │
    │ │ - logs output                  │        │
    │ │ - repeats if needed            │        │
    │ └──────┬─────────────────────────┘        │
    │        │                                  │
    │        ▼                                  │
    │ ┌────────────────┐                        │
    │ │ probeMeta.json │ → updated metadata     │
    │ └────────────────┘                        │
    └───────┬───────────────────────────────────┘
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

### Storage Layout (~/.hypnos/)

    ~/.hypnos/
    ├─ config/   # workflow definitions (*.toml)
    ├─ log/      # logs for each probe (*.log)
    └─ probe/    # metadata for each running probe (*.json)

### Workflow Configuration Example

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

## Installation

### Language-Specific

    Go:  go install github.com/DanielRivasMD/Hypnos@latest

## License

Copyright (c) 2025

See the [LICENSE](LICENSE) file for license details
