# Issue 08: Handoff store — workers write structured handoffs

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/initiative-runner/PRD.md` — see ADR-0004, ADR-0007.

## What to build

Add the **Handoff** entity and the worker side that writes it. When a worker Run finishes an Issue it
records a structured Handoff: what was completed, what was left undone, which commands ran and their exit
codes, and issues discovered. Add a **Handoff store** deep module that serializes/parses a Handoff and
derives `latestState(issueId)` — the durable state the stateless Orchestrator will read on wake.

## Acceptance criteria

- [ ] Handoff table + write path invoked when a worker Run completes
- [ ] Handoff captures done / left-undone / commands+exit-codes / discoveries
- [ ] `latestState(issueId)` derives the current state from an Issue's Handoffs
- [ ] TDD: Handoff (de)serialize + latest-state derivation have failing-first unit tests, then green
- [ ] `go test ./...` and `pnpm test` pass

## Blocked by

- `06-reshape-entities-and-status-claim`

## Comments

### Key decisions

1. **Schema (migration 008)** — `handoff` table with `done text[]`, `left_undone text[]`, `commands jsonb`, `discoveries text[]`, indexed on `issue_id` and `run_id`. Immutable once written (no `updated_at`). FK to `agent_task_queue.id` so the row is co-located with the Run that wrote it.

2. **Pure Go module (`server/internal/handoff/`)** — `LatestState(handoffs []Handoff) State` returns the last Handoff's view; the agent accumulates prior-run history in each Handoff it writes, so the latest entry is the authoritative current state. `SerializeCommands`/`ParseCommands` round-trip `[]CommandResult` to/from JSONB. Zero-dependency (no DB calls) — TDD'd with 10 failing-first unit tests.

3. **Write path** — `HandoffInput` added to `TaskCompleteRequest` and `TaskResult` in daemon types. `writeHandoffOnCompletion` on the handler fires as a best-effort side-effect after `TaskService.CompleteTask` succeeds (same pattern as `advanceInitiativeOnIssueDone`). Guards: only worker Runs (not validators) with a valid `IssueID` and a non-nil input.

4. **`latestState` in TypeScript** — `packages/core/handoff/queries.ts` mirrors the Go derivation: last element wins. `handoffListOptions` TanStack Query options and `handoffKeys` follow the milestone pattern.

5. **API endpoint** — `GET /api/issues/{id}/handoffs` returns rows ordered oldest-first matching `handoff.LatestState`'s expected input order.

6. **Schema validation** — `HandoffSchema` + `ListHandoffsResponseSchema` added to `packages/core/api/schemas.ts` with lenient defaults (arrays fallback to `[]`). 6 new schema tests cover the malformed-response contract.

### Files changed

**Go (new)**
- `server/migrations/008_handoff.{up,down}.sql`
- `server/pkg/db/queries/handoff.sql` (CreateHandoff, ListHandoffsByIssue, GetLatestHandoffByIssue)
- `server/pkg/db/generated/handoff.sql.go` (sqlc generated)
- `server/internal/handoff/handoff.go` — pure domain module
- `server/internal/handoff/handoff_test.go` — 10 unit tests (TDD, failing-first)
- `server/internal/handler/handoff.go` — writeHandoffOnCompletion, ListHandoffs handler
- `server/internal/handler/handoff_test.go` — 4 DB-integrated handler tests

**Go (modified)**
- `server/internal/daemon/types.go` — added `HandoffInput`, `HandoffCommandInput`, `TaskResult.Handoff`
- `server/internal/daemon/client.go` — `CompleteTask` accepts `*HandoffInput`
- `server/internal/daemon/daemon.go` — passes `result.Handoff` to `CompleteTask`
- `server/internal/handler/daemon.go` — added `HandoffInput`, `handoffCommandInput` types; `TaskCompleteRequest.Handoff`; calls `writeHandoffOnCompletion`
- `server/cmd/server/router.go` — registered `GET /api/issues/{id}/handoffs`
- `server/pkg/db/generated/models.go` — sqlc generated Handoff struct

**TypeScript (new)**
- `packages/core/types/handoff.ts` — `Handoff`, `HandoffCommandResult`, `HandoffState`, `ListHandoffsResponse`
- `packages/core/handoff/queries.ts` — `latestState`, `handoffListOptions`, `handoffKeys`
- `packages/core/handoff/index.ts`
- `packages/core/handoff/queries.test.ts` — 4 unit tests for `latestState`

**TypeScript (modified)**
- `packages/core/types/index.ts` — exports for handoff types
- `packages/core/api/schemas.ts` — `HandoffSchema`, `ListHandoffsResponseSchema`, `EMPTY_HANDOFF`, `EMPTY_LIST_HANDOFFS_RESPONSE`
- `packages/core/api/schemas.test.ts` — 6 schema tests
- `packages/core/api/client.ts` — `listHandoffs(issueId)` method
- `packages/core/package.json` — `./handoff` and `./handoff/queries` exports

### Blockers / notes

- The Orchestrator (issue 10) should call `api.listHandoffs(issueId)` and pass the result to `latestState` to derive current issue state on wake.
- The Run/Handoff timeline UI (issue 17) can use `handoffListOptions` directly.
- Physical table rename (`agent_task_queue` → `run`) remains a future mechanical follow-up; `run_id` in the handoff table is the conceptual name but maps to `agent_task_queue.id`.
