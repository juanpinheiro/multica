# Issue 13: Autopilot timezone correctness

**Status:** `done`
**Model:** `sonnet`

## Parent

PRD 1 — Upstream Sync (`.scratch/upstream-sync/PRD.md`).

## What to build

Make autopilot trigger output render in the trigger's own timezone, centralize the default timezone, and fall back to that default when a trigger's configured zone is invalid, ported from upstream. A typo in a zone name should degrade gracefully to the default rather than breaking the trigger.

## Acceptance criteria

- [x] Trigger output renders in the trigger's configured timezone, independent of server timezone.
- [x] The default timezone is defined in one place.
- [x] An invalid configured zone falls back to the default without breaking the trigger.
- [x] Tests cover a valid zone, an invalid zone (fallback), and the centralized default.

## Blocked by

None — can start immediately.

## Comments

### Key decisions

- **`DefaultTimezone` constant in `service/cron.go`.** The single source of truth for the fallback zone (`"UTC"`). Every caller in `autopilot_scheduler.go` that previously held a local `tz := "UTC"` literal now uses `service.DefaultTimezone`, so the default lives in exactly one place.

- **`ResolveTimezone(tz string) *time.Location` helper in `service/cron.go`.** Returns `time.UTC` for an empty or unrecognized zone, logging a warning on invalid input. This satisfies the "graceful fallback without breaking the trigger" requirement — `ResolveTimezone` is always safe to call; callers never need to check for error.

- **`timezone string` added to `DispatchAutopilot`.** The scheduler passes the trigger's stored timezone; webhook, delivery-replay, and manual-trigger call sites pass `""` (defaults to UTC), since webhook triggers carry no timezone. `DispatchAutopilot` resolves the string to `*time.Location` via `ResolveTimezone` and threads it down to `dispatchCreateIssue`.

- **`*time.Location` parameter on `interpolateTemplate` and `buildIssueDescription`.** Both functions now accept a location and use `time.Now().In(loc)` instead of `time.Now().UTC()`. The description timestamp format changed from `"2006-01-02 15:04 UTC"` (hardcoded "UTC" suffix) to `"2006-01-02 15:04 MST"` (Go's zone abbreviation token), so it renders the actual zone abbreviation (e.g. "EST", "EDT", "JST").

- **Scheduler `tz` extracted before dispatch.** In `tickScheduledAutopilots`, the `tz` variable is now computed in the loop body (alongside the autopilot load) before being passed to `DispatchAutopilot`. This mirrors the existing `advanceNextRun` pattern and keeps both computations consistent.

### Files changed

- `server/internal/service/cron.go` — `DefaultTimezone` constant; `ResolveTimezone` function; added `log/slog` import.
- `server/internal/service/cron_test.go` (new) — 4 tests: `TestDefaultTimezone`, `TestResolveTimezone_Valid`, `TestResolveTimezone_Empty`, `TestResolveTimezone_Invalid`.
- `server/internal/service/autopilot.go` — `DispatchAutopilot` gains `timezone string`; `dispatchCreateIssue` gains `loc *time.Location`; `interpolateTemplate` and `buildIssueDescription` accept `*time.Location` and use `time.Now().In(loc)`.
- `server/internal/service/autopilot_test.go` — all `buildIssueDescription`/`interpolateTemplate` calls updated to pass `time.UTC`; new `TestBuildIssueDescription_TimestampUsesZone` and `TestInterpolateTemplate_DateUsesZone` tests.
- `server/cmd/server/autopilot_scheduler.go` — `tz` literals replaced with `service.DefaultTimezone`; `tz` computed in `tickScheduledAutopilots` loop body and passed to `DispatchAutopilot`.
- `server/internal/handler/autopilot.go` — manual-trigger call passes `""`.
- `server/internal/handler/autopilot_webhook.go` — webhook dispatch call passes `""`.
- `server/internal/handler/webhook_delivery.go` — delivery-replay dispatch call passes `""`.
- `server/internal/handler/autopilot_feature_gate_test.go`, `server/internal/handler/handler_test.go`, `server/cmd/server/autopilot_listeners_test.go` — `DispatchAutopilot` calls updated to pass `""`.

### Verification

- `go build ./...` clean.
- `go vet ./...` clean.
- `go test ./internal/service/ ./cmd/server/` — 65 passed.
- `pnpm typecheck` — 4 cached, 0 errors.
- `pnpm test` — 679 TS tests passed.
- Pre-existing Windows-environment failures (git clone on temp paths, local-skill symlink scanning, coalesce-window timing) are unchanged from prior issues.
