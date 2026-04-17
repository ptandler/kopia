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

- [x] Step 1: Create CLI command skeleton
- [ ] Step 2: Add Web API endpoint
- [ ] Step 3: Implement core logic
- [x] Step 4: Add tests (basic test passes)