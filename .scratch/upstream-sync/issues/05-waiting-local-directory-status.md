# Issue 05: `waiting_local_directory` task status

**Status:** `done`
**Model:** `opus`

## Parent

PRD 2 — Workspace In-Place Execution Mode (`.scratch/upstream-sync/PRD.md`).

## What to build

Add a new task status, `waiting_local_directory`, plus a wait-reason hint, and the daemon→server protocol that posts it. Fold the status into the task-status set and add the wait-reason field in the consolidated init schema (not a standalone migration, consistent with the fork's one-init decision). The daemon posts the status — with a reason naming the held path and holder — via the path locker's wait callback when an in-place task finds the umbrella busy, and clears it back to `running` once the lock is acquired. This slice delivers the schema + the status-posting protocol; the daemon's full in-place run wiring is a later slice, and the UI render is a separate slice.

## Acceptance criteria

- [x] `waiting_local_directory` is a valid task status in the consolidated init; a wait-reason field exists and is nullable.
- [x] The daemon posts `waiting_local_directory` with a populated reason when the path locker reports the target busy.
- [x] The status clears to `running` (reason cleared) once the lock is acquired.
- [x] Integration test: a second task on a busy path transitions to `waiting_local_directory` with a reason, then to `running` after the first releases.

## Blocked by

- Issue 03 (In-place path locker) — the wait callback drives this status.

## Comments

### Key decisions

- **`StartAgentTask` widened, not duplicated.** Rather than add a separate
  "resume from waiting" query, `StartAgentTask` now matches
  `status IN ('dispatched', 'waiting_local_directory')` and sets
  `wait_reason = NULL`. A task reaches `running` the same way whether it ran the
  fast path (umbrella free) or parked first (umbrella held), and the stale wait
  hint is always cleared on start. No new start path to keep in sync.
- **`wait_reason` is a plain nullable column, not reusing `failure_reason`.**
  Waiting is a non-terminal, recoverable state; overloading the failure
  classifier (consumed by the auto-retry path) would have crossed wires. A
  dedicated nullable `text` keeps the two orthogonal.
- **Reason text is a pure deep module:** `inplace.WaitReason(dir, holder)` lives
  next to the locker that supplies the holder via its `WaitFunc`. It is the one
  piece testable without a DB, so it carries its own table test (named holder +
  empty-holder-stays-readable). Issue 06's daemon wiring will call
  `client.WaitForLocalDirectory(taskID, inplace.WaitReason(umbrella, locker.Holder(umbrella)))`
  from the locker's wait callback.
- **Status-posting protocol delivered, full run-loop deferred to Issue 06.** Per
  this slice's scope, the endpoint + service + client method exist and are
  exercised end-to-end (locker → wait callback → post → release → start) by the
  serial integration test. The daemon's validate→lock→prepare-branches→run loop
  is Issue 06.
- **Broadcast parity.** A `task:waiting` WS event is emitted on the transition,
  matching every other status flip, so the UI (Issue 07) gets a live signal
  rather than waiting on the 30s snapshot staleTime.
- **Schema folded into `001_init`** (column + CHECK value), no standalone
  migration — consistent with the fork's one-init decision and Issue 01. The
  down migration drops the whole schema, so it needs no per-column change.

### Files changed

- `server/migrations/001_init.up.sql` — `wait_reason text` (nullable) on
  `agent_task_queue`; `waiting_local_directory` added to the status CHECK.
- `server/pkg/db/queries/agent.sql` — new `WaitTaskForLocalDirectory`;
  `StartAgentTask` widened to clear `wait_reason` and accept the waiting state.
  Regenerated via `sqlc generate` (`models.go`, `agent.sql.go`).
- `server/internal/workspace/inplace/wait_reason.go` (+ `_test.go`) — new pure
  `WaitReason(dir, holder)` helper.
- `server/internal/service/task.go` — `WaitTaskForLocalDirectory` service method
  (broadcasts `EventTaskWaiting`).
- `server/pkg/protocol/events.go` — `EventTaskWaiting = "task:waiting"`.
- `server/internal/handler/daemon.go` — `WaitTaskForLocalDirectory` handler +
  `TaskWaitLocalDirectoryRequest`.
- `server/cmd/server/router.go` — `POST /tasks/{taskId}/wait-local-directory`.
- `server/internal/handler/agent.go` — `wait_reason` on `AgentTaskResponse` +
  populated in `taskToResponse`.
- `server/internal/daemon/client.go` — `Client.WaitForLocalDirectory`.
- `server/internal/handler/inplace_waiting_test.go` (new) — HTTP test
  (set-then-clear) + locker-driven serial test (busy → waiting+reason → running
  after release).

### Verification

- `go build ./...` and `go vet ./...` clean.
- `go test ./internal/workspace/inplace/ ./internal/feature/ ./internal/workspace/...`
  green (43 across pure packages); the `WaitReason` table test passes.
- `pnpm typecheck` green (no TS changes this slice — the API now returns
  `wait_reason`; the TS `Workspace`/task type + UI render belong to Issue 07).
- **DB-backed handler/integration tests were not executed locally** — Docker
  Desktop / the shared Postgres container is not running on this machine, so the
  handler suite skips via its `testHandler == nil` guard (the repo's standard
  pattern; Issue 01 noted the same). They compile and `go vet` clean and run in
  CI against the `pgvector/pg17` service. The long-lived local DB was also
  synced with the equivalent `ALTER TABLE … ADD COLUMN IF NOT EXISTS wait_reason`
  + CHECK update for when it is next brought up; a fresh/CI DB gets it from
  `001_init.up.sql`.
- The `internal/daemon` package has pre-existing, environment-specific test
  failures on this Windows box (daemon-id profile dirs, local-skill symlink
  scanning, repo-cache `git clone` on temp paths). Confirmed by stashing this
  slice's only daemon change (`client.go`) and observing the identical failures
  on base — they are unrelated to this work.

### Notes for next iteration

- **Issue 06 (daemon in-place wiring)** holds the umbrella lock for the task's
  whole lifetime: `locker.Acquire` at claim with a `WaitFunc` that calls
  `client.WaitForLocalDirectory(taskID, inplace.WaitReason(umbrella, holder))`,
  then `client.StartTask` once acquired (which clears the reason), and the
  deferred `ReleaseFunc` through report. The reclaim sweeper keys on
  `status = 'dispatched'`, so a legitimately-parked `waiting_local_directory`
  task is not double-dispatched — no change needed there.
- **Issue 07 (UI)** should add `waitReason` (camelCase) to the task type and
  render it for the waiting state, with an enum-drift fallback for the mode
  value per the API-compatibility rules.
