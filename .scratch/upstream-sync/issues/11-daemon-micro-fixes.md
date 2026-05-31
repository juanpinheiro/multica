# Issue 11: Daemon micro-fixes

**Status:** `done`
**Model:** `sonnet`

## Parent

PRD 1 — Upstream Sync (`.scratch/upstream-sync/PRD.md`).

## What to build

Three small, independent daemon reliability fixes ported from upstream, grouped because they share the daemon area and test approach. Confirm each is absent locally before porting.

1. **Codex Desktop bundle CLI detection** — extend the daemon's Codex CLI discovery to recognize a Codex Desktop bundle location, so Codex works without hand-wiring its path.
2. **Terminal task callback retries** — wrap the daemon's terminal callback (task→server result report) in a bounded retry on transient (network / 5xx) failures, so a finished task is not stranded by a momentary reporting failure.
3. **GC empty-parent-id guard** — in the daemon's garbage-collection pass, skip metadata rows whose parent ids are empty rather than acting on them.

## Acceptance criteria

- [x] Codex Desktop bundle CLI is detected and used when present.
- [x] Terminal callbacks retry on transient errors with a bounded number of attempts; a permanent failure still surfaces.
- [x] GC skips meta rows with empty parent ids without error.
- [x] Tests cover each fix's failure mode.

## Blocked by

None — can start immediately.

## Comments

### Key decisions

- **Codex Desktop bundle detection.** Added `codexDesktopBundleCandidates()` (returns platform-specific app-bundle paths — `~/Applications/Codex.app/…` on macOS, `%LOCALAPPDATA%\Programs\codex\codex.exe` on Windows) and `probeBundle(candidates, model)` that tries each via `exec.LookPath`. Wired as a second fallback in `LoadConfig` after the standard PATH/shell-fallback probe for "codex" misses. A `codexBundleCandidatesForTest` override variable lets tests inject a fake path without touching the real filesystem.

- **Terminal callback retries.** Added `isTransientCallbackError(err)` (true for non-`requestError` network failures and 5xx; false for nil, context cancellation, and 4xx), `retryTerminalCallback(ctx, fn, log)` (retries up to `maxTerminalCallbackAttempts=3` with `terminalCallbackRetryDelay` between attempts; delay is a `var` so tests can set it to 0). `reportTaskResult` now wraps all three call sites (`CompleteTask`, fallback `FailTask`, and non-completion `FailTask`) in `retryTerminalCallback`.

- **GC empty-parent-id guard.** Added a 3-line `if meta.<ID> == ""` early-return guard at the top of each `gcDecision*` function (`gcDecisionIssue`, `gcDecisionChat`, `gcDecisionAutopilotRun`, `gcDecisionQuickCreate`). Returns `gcActionSkip` and logs a warning. Prevents making API calls with empty path segments like `/api/daemon/issues//gc-check` when a corrupted or malformed meta file has an empty ID.

### Files changed

- `server/internal/daemon/config.go` — `runtime` import; `codexBundleCandidatesForTest` override var; `codexDesktopBundleCandidates()`; `probeBundle()`; codex probe falls through to bundle detection.
- `server/internal/daemon/config_test.go` — `TestProbeBundle_*` (3 cases), `TestCodexDesktopBundleCandidates_AllAbsolute`, `TestLoadConfig_UsesCodexDesktopBundle`.
- `server/internal/daemon/daemon.go` — `maxTerminalCallbackAttempts`, `terminalCallbackRetryDelay` (var), `isTransientCallbackError()`, `retryTerminalCallback()`; `reportTaskResult` updated to use retry wrapper.
- `server/internal/daemon/daemon_test.go` — `TestIsTransientCallbackError_*` (5 cases), `TestReportTaskResult_RetriesOnTransient`, `TestReportTaskResult_GivesUpAfterMaxAttempts`.
- `server/internal/daemon/gc.go` — empty-ID guard in each of the four `gcDecision*` functions.
- `server/internal/daemon/gc_test.go` — `TestShouldCleanTaskDir_EmptyIssueID_Skip`, `TestShouldCleanTaskDir_EmptyChatSessionID_Skip`, `TestShouldCleanTaskDir_EmptyAutopilotRunID_Skip`, `TestShouldCleanTaskDir_EmptyTaskID_Skip`.

### Verification

- `go build ./...` + `go vet ./internal/daemon/...` clean.
- 21 new tests pass (`TestShouldCleanTaskDir_Empty*`, `TestIsTransientCallbackError_*`, `TestReportTaskResult_*`, `TestProbeBundle_*`, `TestCodexDesktopBundleCandidates_*`, `TestLoadConfig_UsesCodexDesktopBundle`).
- `pnpm typecheck` + `pnpm test` green (679 TS tests, 82 test files).
- Pre-existing environment-specific failures on this Windows box (`local_skills_test.go`, `runtime_gone_test.go`, `identity_test.go`) are unrelated — confirmed pre-existing in prior issue comments.
