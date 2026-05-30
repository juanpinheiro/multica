# Issue 12: Autopilot per-trigger webhook event filters

**Status:** `done`
**Model:** `sonnet`

## Parent

PRD 1 — Upstream Sync (`.scratch/upstream-sync/PRD.md`).

## What to build

Add the ability to filter, per webhook trigger, which event types fire an autopilot, ported from upstream. The schema change folds into the consolidated init (not a standalone migration). The matching decision — does an incoming event match a trigger's filter? — is extracted as a pure module so it can be tested in isolation. An event outside a trigger's filter is recorded as a skip through the existing autopilot-run audit, not silently dropped; an absent/empty filter preserves the current fire-on-everything behavior.

## Acceptance criteria

- [x] A webhook trigger can declare an event-type filter; schema folded into the consolidated init.
- [x] Pure matcher module decides match/skip given an event and a trigger's filter; table-driven tests cover in-filter, out-of-filter, and empty/absent filter (fires on everything).
- [x] An out-of-filter event is recorded as a skip in the autopilot-run audit; an in-filter event dispatches as today.
- [x] Handler test exercises the end-to-end filter behavior over HTTP.

## Blocked by

None — can start immediately.

## Comments

### Key decisions

- **Pure `eventfilter` package** at `server/internal/autopilot/eventfilter/`. Single exported function `Match(event, filters)` using `path.Match` for glob support (`github.pull_request.*` matches `github.pull_request.opened`). Empty/nil filter returns true (fire-on-everything, backward compatible). Malformed patterns are silently skipped.
- **`event_filters text[]` column with `DEFAULT '{}'`** on `autopilot_trigger` in the consolidated `001_init.up.sql`, per the one-init rule. No separate migration needed; an empty array is the "no filter" default.
- **Dedicated `SetAutopilotTriggerEventFilters` query** mirrors the `SetAutopilotTriggerSigningSecret` pattern — a focused write-only endpoint so the array update is clean and separate from the general `UpdateAutopilotTrigger` path. The `UpdateAutopilotTrigger` handler calls it when `event_filters` is non-nil in the PATCH body.
- **Filter check in the webhook handler (step 10.5)** — placed after all gate checks (signature, trigger enabled, autopilot status) and before `DispatchAutopilot`. On mismatch, calls `AutopilotService.RecordEventFilterSkip` which reuses the existing private `recordSkippedRun` helper with reason `"event_filter_mismatch"`, then updates the delivery to `dispatched` (with the run id) and returns `200 skipped`. This mirrors exactly how admission-skipped runs work elsewhere.
- **`nonNilStringSlice` helper** in the handler ensures `event_filters` always serializes as `[]` not `null` in JSON responses.

### Files changed

- `server/internal/autopilot/eventfilter/filter.go` (new) — `Match` function
- `server/internal/autopilot/eventfilter/filter_test.go` (new) — 9 table-driven test cases
- `server/migrations/001_init.up.sql` — `event_filters text[] DEFAULT '{}'::text[] NOT NULL` on `autopilot_trigger`
- `server/pkg/db/queries/autopilot.sql` — new `SetAutopilotTriggerEventFilters` query
- `server/pkg/db/generated/models.go` — `EventFilters []string` on `AutopilotTrigger`
- `server/pkg/db/generated/autopilot.sql.go` — `event_filters` in all trigger SELECT/RETURNING lists and Scan calls; `ClaimDueScheduleTriggersRow`, `GetWebhookTriggerByTokenRow`, `RecoverLostTriggersRow` structs updated; new `SetAutopilotTriggerEventFilters` function
- `server/internal/handler/autopilot.go` — `EventFilters []string` on `AutopilotTriggerResponse`; `nonNilStringSlice` helper; `EventFilters *[]string` on `UpdateAutopilotTriggerRequest`; handler calls `SetAutopilotTriggerEventFilters` when field is non-nil
- `server/internal/handler/autopilot_webhook.go` — `eventfilter` import; step 10.5 event-filter check before dispatch
- `server/internal/service/autopilot.go` — `RecordEventFilterSkip` public method
- `server/internal/handler/autopilot_webhook_handler_test.go` — `setTriggerEventFilters` helper; 4 new tests: match dispatches, mismatch records skip, empty filter fires on all, glob pattern matches

### Verification

- `go build ./...` clean
- `go vet ./...` clean
- `go test ./internal/autopilot/...` — 10 passed (all eventfilter cases)
- `pnpm typecheck` — 4 cached, 0 errors
- DB-backed handler tests (event filter end-to-end) compile and vet clean; run against the pgvector/pg17 CI service
