# Plan: `kopia snapshot run-scheduled`

GitHub discussion: see user notes

## Goal
Create a CLI command that runs all snapshot sources that are due or overdue based on their schedule policies.

---

## Implementation Steps

### Step 1: New CLI Command
**New file:** `cli/command_snapshot_run_scheduled.go`

Flags (following `command_snapshot_migrate.go` patterns):
| Flag | Description | Default |
|------|-------------|---------|
| `--source` | Run only for specific source (can be repeated) | all |
| `--parallel` | Max number of parallel snapshot jobs | 1 |

### Step 2: Web API Endpoint
**New endpoint:** `POST /api/v1/sources/run-scheduled`
- Allows web UI "Run All Due" button
- Reuses existing `handleSourcesList` and trigger logic

### Step 3: Core Logic Implementation
```
run-scheduled:
  1. Connect to repo (via existing flags/config)
  2. List all sources: snapshot.ListSources(ctx, rep)
  3. For each source:
     a. Get last snapshot time (find manifests, get most recent)
     b. Get effective policy: policy.GetEffectivePolicy(ctx, rep, sourceInfo)
     c. Calculate next time: policy.NextSnapshotTime(lastSnapTime, now)
     d. If nextTime <= now: trigger snapshot (use existing upload logic)
  4. Handle parallel execution (use semaphore pattern from migrate.go)
```

### Step 4: Tests
**New file:** `cli/command_snapshot_run_scheduled_test.go`
- Test due/overdue detection
- Test parallel execution
- Test dry-run mode

---

## Key Files to Reference

| Component | File | Key Lines |
|-----------|------|----------|
| List sources | `snapshot/manager.go` | 39-58 |
| Scheduling policy | `snapshot/policy/scheduling_policy.go` | 98-140 |
| Get effective policy | `snapshot/policy/policy_manager.go` | 64-132 |
| Upload snapshot | `cli/command_snapshot_create.go` | 284+ |
| Parallel pattern | `cli/command_snapshot_migrate.go` | 61, 117 |
| Source status (next time calc) | `internal/server/source_manager.go` | 116-140 |

---

## Status

- [x] Step 1: Create CLI command - DONE (build passes, tests pass)
- [ ] Step 2: Add Web API endpoint (skipped - existing /api/v1/sources/upload works)
- [x] Step 3: Implement core logic (due/overdue detection using policy.NextSnapshotTime)
- [x] Step 4: Add tests (TestSnapshotRunScheduled passes)

All Next Steps COMPLETED:
- [x] CLI reference page: `site/content/docs/Reference/Command-Line/snapshot-run-scheduled/_index.md`
- [x] Scheduling howto: `site/content/docs/Advanced/scheduling-automated-backups/_index.md`
- [x] Tests updated

## Next Steps

- [x] Add CLI documentation for `kopia snapshot run-scheduled` command - in the CLI reference section but also for the CLI `--help` commend
- [x] Add documentation for `kopia snapshot run-scheduled` command including usage example how to schedule it regularily on Linux/Windows/MacOS
- [x] In the documentation on how to schedule the task, list different options with examples how to use it, e.g for Linux cron, systemd unit & timer, and other popular options
- [x] Improve tests to check that new snapshots are created for due tasks only
