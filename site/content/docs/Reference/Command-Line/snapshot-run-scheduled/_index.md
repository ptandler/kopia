---
title: "kopia snapshot run-scheduled"
linkTitle: "snapshot run-scheduled"
weight: 30
---

`kopia snapshot run-scheduled` runs all snapshot sources that are due or overdue based on their schedule policies.

## Synopsis

```shell
kopia snapshot run-scheduled [<flags>]
```

## Description

The `run-scheduled` command checks each configured snapshot source against its schedule policy and runs snapshots that are due or overdue. It respects the scheduling policy defined for each source, including intervals, times of day, and day-of-week settings.

This command is useful for:
- Running automated backups on a schedule (via cron, systemd timers, or task schedulers)
- Ensuring all due snapshots are taken without manually specifying each source
- Running multiple snapshots in parallel

## Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--source` | Source path to run (can be repeated). If not specified, runs all configured sources. | all sources |
| `--parallel` | Number of snapshots to run in parallel. | 1 |
| `--dry-run` | Show what would run without executing. | false |

## Examples

### Run all scheduled snapshots

```shell
kopia snapshot run-scheduled
```

### Run a specific source

```shell
kopia snapshot run-scheduled --source /home/user/documents
```

### Run multiple sources in parallel

```shell
kopia snapshot run-scheduled --parallel=4
```

### Preview what would run (dry-run)

```shell
kopia snapshot run-scheduled --dry-run
```

## Scheduling

To run snapshots automatically on a schedule, add this command to your system's task scheduler:

- **Linux**: Use cron or systemd timers
- **macOS**: Use launchd or cron
- **Windows**: Use Task Scheduler

See [Scheduling Automated Backups](../../../advanced/scheduling-automated-backups/) for detailed examples.