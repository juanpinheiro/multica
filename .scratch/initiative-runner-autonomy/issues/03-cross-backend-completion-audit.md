# Issue 03: Cross-backend completion-robustness audit

**Status:** `done`
**Model:** `opus`

## Parent

`.scratch/initiative-runner-autonomy/PRD.md` — Module #1 (user story 4).

## What to build

Issue 02 fixes the completion hang in the claude backend by making the authoritative completion message drive a proactive teardown. The same class of hang exists in any backend whose lifecycle is driven solely by a `for scanner.Scan()` reader loop with no proactive teardown on its own completion signal — the process can finish its work, emit a final message, and then fail to exit, leaving the reader blocked until the idle watchdog.

Audit the other agent backends (e.g. gemini, copilot, and any others) and apply the same completion-driven teardown where the same hang is possible. Backends that already drive teardown from a semantic completion signal (codex) need no change. The audit must use judgment per backend — each has a different completion signal and shutdown sequence; the fix is to identify that signal and trigger teardown on it without breaking the backend's normal exit path.

## Acceptance criteria

- [ ] Every agent backend is reviewed; the audit records which are reader-loop-only (vulnerable), which already drive teardown (safe), and the rationale for each.
- [ ] Each vulnerable backend delivers its `Result` with the correct disposition within a short grace window after its authoritative completion signal, instead of blocking until the idle watchdog.
- [ ] No backend's normal prompt-exit fast path is regressed.
- [ ] The idle watchdog remains unchanged as the last-resort net for runs that never signal completion.
- [ ] Each fixed backend has a regression test mirroring the claude one (completion signal then hang → completed within grace).
- [ ] `make check` passes (known Windows `repocache` git-clone flakes excepted).

## Blocked by

- Issue 02 (establishes the teardown pattern this audit replicates).

## Comments

### Audit: every backend reviewed

Two structural shapes exist in `server/pkg/agent/`. The hang is only possible in
the first.

**Shape A — single reader goroutine that keeps scanning after the completion
message and ends only on stdout EOF / `cmd.Wait`.** Vulnerable: a CLI that emits
its authoritative completion message then fails to exit blocks the reader until
the 30-minute idle watchdog and is mislabelled `blocked`.

**Shape B — a separate lifecycle goroutine that blocks on the completion signal
and, the moment it fires, closes stdin + cancels the `exec.CommandContext` run
context (killing the process → stdout EOF → reader unblocks).** Safe: this is
exactly the codex `cancel-on-turn-complete` pattern the PRD names as needing no
change.

| Backend | Shape | Completion signal | Verdict |
|---|---|---|---|
| claude | A | `result` msg | already fixed (issue 02), now uses shared helper |
| codex | B | `turnDone` (turn/completed) | SAFE — no change (`codex.go:370-371`) |
| hermes | B | `session/prompt` RPC response | SAFE — `c.request` returns → `cancel()` (`hermes.go:302,344-345`) |
| kimi | B | `session/prompt` (reuses `hermesClient`) | SAFE — `kimi.go:277,312-313` |
| kiro | B | `session/prompt` (reuses `hermesClient`) | SAFE — `kiro.go:272,306-307` |
| gemini | A | `result` event | **FIXED** |
| antigravity | A | `result` event | **FIXED** |
| copilot | A | synthetic `result` event | **FIXED** |
| cursor | A | `result` event | **FIXED** |
| pi | A | `turn_end` event | **FIXED** |
| opencode | A | *none in-stream* | **NOT fixable via this pattern** — see below |
| openclaw | A | parsed result, but `io.ReadAll` path | **NOT fixable via this pattern** — see below |

### What was changed

- **`server/pkg/agent/teardown.go` (new)** — shared, reusable pieces extracted
  so the fix is applied once, not copy-pasted per backend:
  `defaultResultTeardownGrace` (5s), `resolveResultTeardownGrace(override)`
  (test seam), `scheduleResultTeardown(ctx, cancel, grace)` (grace-then-cancel
  goroutine, exits early if the process leaves on its own).
- **`claude.go`** — refactored onto the shared helper; removed its private
  `claudeResultTeardownGrace` const, `teardownGrace()` method, and
  `scheduleClaudeTeardown` func (were parallel abstractions). Behaviour
  unchanged; its three issue-02 tests still pass.
- **gemini / antigravity / copilot / cursor / pi** — each got a
  `resultTeardownGrace` field (test seam), a `resultSeen` guard that schedules
  the teardown once on the authoritative completion message, and the post-loop
  disposition block wrapped in `if !resultSeen { … }` so the teardown cancel (or
  the non-zero exit from the force-kill that follows) can't clobber a real
  `completed`/`failed` into `aborted`/`failed`. Mirrors claude exactly.
- **`teardown_test.go` (new)** — one regression test per fixed backend: a fake
  CLI emits its completion message then `exec sleep 30` (the process itself holds
  the stdout pipe open). Each asserts the run resolves `completed` within the
  injected 200ms grace — far under the watchdog — never `aborted`/`timeout`.

### Two vulnerable backends deliberately left to the watchdog

Per the issue's "use judgment per backend" instruction, two Shape-A backends have
**no authoritative completion message arriving before EOF**, so the
completion-driven teardown pattern does not apply and a forced fix would be a
net regression:

- **opencode** — the happy-path stream ends with a `step_finish` whose part
  carries no terminal `reason` (`opencode_test.go:509`), and intermediate steps
  emit the same event. There is no distinct "done" signal to act on; EOF *is* the
  completion signal. Gating teardown on `step_finish` would risk tearing down a
  still-running multi-step agent (worse than the hang). Idle watchdog remains the
  net.
- **openclaw** — `processOutput` reads via `io.ReadAll(stdout)` as its primary
  fast path (`openclaw.go:122`), which is inherently EOF-bound — it cannot
  observe a completion message before the stream ends. Applying the pattern would
  require abandoning the whole-buffer fast path (a risky refactor, out of scope).
  Idle watchdog remains the net.

### Test results / notes

- All new tests + the full `./pkg/agent` suite pass on Linux (cross-compiled the
  test binary with the Windows toolchain — `GOOS=linux go test -c` — and ran it
  under WSL Ubuntu, since the process-spawning fixtures are `#!/bin/sh` and skip
  on native Windows). Each fixed backend resolved in ~201ms (the 200ms grace),
  confirming teardown fires well under the watchdog.
- `go build ./...` and `go vet ./pkg/agent` clean on Windows.
- Idle watchdog (30-minute window + in-flight-tool guard in `daemon.go`) left
  unchanged, as required.
- No blockers for the next iteration.
