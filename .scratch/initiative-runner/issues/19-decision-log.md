# Issue 19: Decision Log — the self-evolving architecture memory

**Status:** `done`
**Model:** `opus`

## Parent

`.scratch/initiative-runner/PRD.md` — see CONTEXT.md (Decision Log / Architecture Memory).

## What to build

A self-evolving layer of architectural decisions, kept current by agents. Add a Decision Log (a table plus
the existing `docs/adr/` files) and a "retrospective" agent run that, at Milestone/Initiative boundaries,
revisits technical decisions, records what was learned, and updates the ADRs and `CONTEXT.md`. This closes
the "system improves itself" loop.

Model `opus`: novel design and it edits architectural docs — judgment matters.

## Acceptance criteria

- [ ] Decision Log table + write path; entries link to ADRs / CONTEXT terms
- [ ] A retrospective agent runs at an Initiative boundary and records decisions/learnings
- [ ] Retrospective updates `docs/adr/` and/or `CONTEXT.md` when a decision changes
- [ ] `go test ./...` and `pnpm test` pass

## Blocked by

- `06-reshape-entities-and-status-claim`

## Comments

### Key decisions

1. **Retrospective is a third Run `role` (`worker | validator | retrospective`).** It mirrors the
   validator path exactly — a fresh-session Run dispatched at a boundary, a synthesized skill +
   prompt injected via execenv, and a structured XML-style output block parsed on completion — so
   the Decision Log rides the existing dispatch/claim/complete plumbing with no new execution path.
   Migration 011 widens the `agent_task_queue_role_check` constraint to allow the new role.

2. **Pure deep module `server/internal/decisionlog` (the TDD focus, 10 failing-first → green).**
   `Parse(output)` scans a `<multica-decision-log>` block and decodes `{"entries":[...]}` (nil on
   missing/empty/malformed, mirroring the validator's parse); `ValidEntries(*Output)` trims, dedupes
   the ref/term lists, and drops entries missing a title or decision. Both the daemon (parse) and the
   handler (persist) import it, so the parsing/normalization logic has a single home.

3. **Trigger is the Initiative boundary, not every Milestone.** The Orchestrator dispatches the
   retrospective from `orchestrateIssue` exactly when it advances the Initiative to `in_review`
   (`Decide` only returns `AdvanceTo=in_review` once, from ready/running), right beside
   `notifyInitiativeReadyForReview`. `dispatchRetrospective` guards with
   `CountActiveRetrospectiveRunsByFeature` so a racing reconcile can't double-dispatch.
   `onTaskCompleted` now skips retrospective Runs (as it already skipped validators), and
   `recordRetrospectiveOnCompletion` does NOT re-enter the orchestrator (the Initiative is already
   parked at `in_review`), so there is no reconcile loop.

4. **Docs are updated by the agent, not the server.** The retrospective skill/prompt instruct the
   Agent to edit `docs/adr/` and `CONTEXT.md` in place where a decision changed (the file-editing is
   the Agent's job, same posture as the validator's sub-agent fan-out). The persisted entries carry
   `adr_refs` (ADR numbers) and `context_terms` (CONTEXT glossary terms) as the durable links.
   CONTEXT.md's "Decision Log" glossary entry was firmed up (dropped "(Name tentative.)") to record
   the now-built `decision_log` table + retrospective role.

5. **Read surface + API-compatibility rule honored.** `GET /api/features/{id}/decisions` lists an
   Initiative's decisions newest-first; the TS client runs the response through
   `DecisionLogEntrySchema`/`parseWithFallback` (lenient defaults, malformed-response tests added).
   `decisionLogOptions` query options feed the Decision Log view (issue 18's deferred UI).

### Files changed

**Go (new)**
- `server/migrations/011_decision_log.{up,down}.sql`
- `server/internal/decisionlog/{decisionlog.go,decisionlog_test.go}` — pure Parse/ValidEntries (TDD)
- `server/pkg/db/queries/decision_log.sql` (+ regenerated `generated/decision_log.sql.go`, `models.go`)
- `server/internal/handler/{decision_log.go,decision_log_test.go}` — record/dispatch/list + DB tests
- `server/internal/daemon/{retrospective.go,retrospective_test.go}` — skill + prompt

**Go (modified)**
- `server/pkg/db/queries/agent.sql` — `CreateRetrospectiveTask`, `CountActiveRetrospectiveRunsByFeature`
- `server/internal/service/task.go` — `DispatchRetrospectiveRun`
- `server/internal/handler/daemon.go` — `TaskCompleteRequest.Retrospective`, call recorder
- `server/internal/handler/orchestrator.go` — dispatch retrospective at `in_review`; skip role in `onTaskCompleted`
- `server/internal/daemon/{daemon.go,prompt.go,types.go,client.go}` — role dispatch, parse, forward
- `server/cmd/server/router.go` — `GET /api/features/{id}/decisions`
- `CONTEXT.md` — Decision Log glossary entry firmed up

**TypeScript**
- `packages/core/types/decision-log.ts` (+ `types/index.ts` export)
- `packages/core/api/schemas.ts` (+ `schemas.test.ts`) — `DecisionLogEntrySchema`, list schema, empties, malformed tests
- `packages/core/api/client.ts` — `listDecisionLog`
- `packages/core/decision-log/queries.ts` — `decisionLogOptions`
- `packages/core/package.json` — `./decision-log/queries` export

### Blockers / notes

- `pnpm typecheck` clean; `pnpm test` green (408 core + 706 views, incl. new schema tests);
  `go test` green for every touched package (`internal/decisionlog`, `internal/handler`,
  `internal/service`, `internal/orchestrator`, and the daemon retrospective tests); `go vet ./...`
  and `go build ./...` clean. Migration 011 applied to the shared dev/test DB.
- Remaining full-suite Go failures (daemon Cursor skill-bundle + runtime-gone timing, execenv
  openclaw tilde, repocache Windows git-clone, pkg/agent missing opencode/kiro/kimi/hermes binaries,
  pkg/redact home-dir) are all pre-existing and environment-specific, unrelated to this change.
- Token/wall-clock per-Run tracking is still not recorded (noted in issue 11); the retrospective
  carries no budget input and doesn't need one.
- The retrospective is attached to the boundary-triggering Issue's assignee agent for runtime
  resolution; a future change could resolve a distinct reviewer-style agent (as the validator does)
  if creator-verifier separation on the retrospective becomes desirable.
