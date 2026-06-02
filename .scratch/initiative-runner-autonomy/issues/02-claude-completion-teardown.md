# Issue 02: Completion-driven teardown in the claude backend

**Status:** `done`
**Model:** `opus`

## Parent

`.scratch/initiative-runner-autonomy/PRD.md` — Module #1. This is the highest-value blocker for multi-hour AFK runs.

## What to build

The claude backend drives its whole lifecycle from a stdout reader loop and only delivers its `Result` once the process exits (stdout EOF → `cmd.Wait`). When the `claude` CLI emits its authoritative `result` message (it has genuinely finished) but the process does NOT exit — a CLI hang in stream-json mode, or a grandchild process holding the stdout pipe open — the reader never sees EOF, no `Result` is produced, and the daemon blocks until the 30-minute idle watchdog. On a serial-within Initiative that single hung task freezes the entire pipeline, and the watchdog then mislabels the successful run as `blocked`/`idle_watchdog`.

Make the `result` message — not process exit — the authoritative completion signal, mirroring the codex backend (which closes stdin and cancels its context the moment its turn-complete signal lands, precisely "so the long-running process doesn't keep stdout open and block the reader forever").

Behavior:
- When the reader receives the `result` message, capture the final disposition there (`completed`, or `failed` when `is_error`), along with output, session id, and usage (as today), and record that a result has been seen.
- After a result is seen, give the process a short grace window (~5s) to exit cleanly; if it hasn't, cancel the run context. Cancellation closes stdout (unblocking the scanner) and lets `cmd.Wait()` return; the existing `WaitDelay` force-closes any grandchild still holding the pipe.
- A run that emitted a `result` resolves as `completed`/`failed` regardless of whether the process then hung. It must NOT be routed through the `idle_watchdog` → `blocked` disposition.
- The idle watchdog (window and in-flight-tool guard) is unchanged — it remains the safety net for runs that never emit a result at all. Do NOT shorten it.

## Acceptance criteria

- [ ] A scripted/fake claude process that emits a `result` then does not exit resolves as `completed` (or `failed` when `is_error`) within the grace window — well under the idle-watchdog horizon — and is NOT dispositioned `idle_watchdog`/`blocked`.
- [ ] A process that exits promptly after `result` is unaffected (fast path still works, output/session/usage intact).
- [ ] The idle-watchdog code path and its 30-minute default are unchanged.
- [ ] Regression tests in `claude_test.go` cover both the hung-after-result and exits-promptly cases. Prior art: existing `claude_test.go` stream-driven tests and the codex teardown tests.
- [ ] `make check` passes (known Windows `repocache` git-clone flakes excepted).

## Blocked by

None - can start immediately.

## Comments

### Key decisions

- **The `result` message drives teardown, not process exit.** When the stdout
  reader receives the `result` message it now records `resultSeen = true` (in
  addition to capturing disposition/output/session/usage as before) and schedules
  a proactive teardown via `scheduleClaudeTeardown(runCtx, cancel, grace)`.
- **Grace-then-cancel.** `scheduleClaudeTeardown` runs a goroutine that waits the
  grace window (default `claudeResultTeardownGrace = 5s`) and then cancels the run
  context. Cancellation trips the existing `<-runCtx.Done() → stdout.Close()`
  goroutine, which unblocks `scanner.Scan()`; `cmd.WaitDelay` (10s) force-closes
  any grandchild still holding the pipe. If the process exits on its own first,
  the goroutine observes `ctx.Done()` and exits without cancelling — the fast path
  is untouched. This mirrors codex's cancel-on-turn-complete teardown.
- **Disposition correctness.** The post-loop reclassification block
  (`DeadlineExceeded → timeout`, `Canceled → aborted`, `exitErr → failed`) is now
  guarded by `if !resultSeen`. A run that emitted a result keeps the
  completed/failed disposition the result carried, regardless of the teardown
  cancel or the non-zero exit from the force-kill that follows it. So a
  result-bearing run is never routed through `idle_watchdog`/`blocked`.
- **Idle watchdog unchanged.** No change to the 30-minute window or its in-flight
  guard (it lives in `daemon.go`); it remains the safety net for runs that never
  emit a result at all. Our `cancel()` cancels `runCtx` (a child of the daemon's
  `agentCtx`), not `agentCtx`, so the daemon's `idleWatchdogFired` flag stays
  false and the result flows up as completed.
- **Test seam.** Added unexported `resultTeardownGrace` field + `teardownGrace()`
  accessor on `claudeBackend` so tests inject a 200ms grace (same-package
  construction). Zero means use the 5s default. Not a compat shim — just keeps the
  hung-after-result tests fast and deterministic.

### Files changed

- `server/pkg/agent/claude.go` — `claudeResultTeardownGrace` const, `resultSeen`
  tracking + teardown trigger in the `result` case, `!resultSeen` guard on the
  disposition block, `scheduleClaudeTeardown` helper, `resultTeardownGrace` field.
- `server/pkg/agent/claude_test.go` — three new tests: hung-after-result resolves
  `completed` within grace (fixture uses `exec sleep` so the process itself hangs
  with the pipe open); error-result-then-hang resolves `failed` with the result's
  message; prompt-exit-after-result fast path unaffected and does not wait the grace.

### Test results / notes

- Verified on Linux (Go 1.26.1 in WSL, since the process-spawning fixtures are
  `#!/bin/sh` and skip on native Windows): the three new tests pass in ~0.2s,
  full `./pkg/agent` passes (5.0s), `go build ./...` and `go vet ./pkg/agent` clean.
- Implementation detail worth knowing for issue 03 (cross-backend audit): a real
  CLI hang where the spawned process itself holds the pipe resolves within the
  grace window (~ms after cancel). A hang where an orphaned *grandchild* inherits
  the stdout pipe resolves in grace + `cmd.WaitDelay` (~10s) — still seconds, not
  the 30-minute watchdog. Both are bounded; the audit should replicate the same
  cancel-on-completion-signal pattern per backend.
