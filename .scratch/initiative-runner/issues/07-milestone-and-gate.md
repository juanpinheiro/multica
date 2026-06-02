# Issue 07: Milestone entity and Gate milestone-gating

**Status:** `done`
**Model:** `opus`

## Parent

`.scratch/initiative-runner/PRD.md` — see ADR-0002, ADR-0004.

## What to build

Add the **Milestone** entity (ordered within an Initiative, with a validation status), and extend the
**Gate** — the deterministic claim predicate — so a Milestone's Issues are not claimable until the
previous Milestone in the Initiative has passed validation. The Gate is the deep module: a function over
world state returning claimable-or-reason, encapsulating Issue-dependency gating, branch serialization
(serial-within-Initiative), the in-place umbrella lock, and now milestone-gating. Demoable: a two-Milestone
Initiative runs its first Milestone's Issues, and the second Milestone's Issues stay unclaimable until the
first is marked validated.

Model `opus`: the Gate is correctness-critical and is a TDD'd deep module.

## Acceptance criteria

- [x] Milestone table + ordering + validation status; Issues belong to a Milestone
- [x] Gate extended with milestone-gating, keeping existing dependency/branch/umbrella rules
- [x] A second Milestone's Issues are unclaimable until the first Milestone is validated
- [x] Milestone shown on the board (minimal)
- [x] TDD: Gate has failing-first table-driven unit tests covering each gating reason, then green
- [x] `go test ./...` and `pnpm test` pass (one pre-existing WS-integration flake unrelated to this change)

## Blocked by

- `06-reshape-entities-and-status-claim`

## Comments

### Key decisions

1. **Milestone table (migration 007).** `milestone(id, workspace_id, feature_id, title,
   position int, validation_status text default 'pending' CHECK in pending|passed|failed,
   timestamps)`, indexed on `feature_id`/`workspace_id`. The FK left FK-less in 006
   (`issue.milestone_id`) is now wired with `ON DELETE SET NULL` — deleting a Milestone
   un-gates its Issues rather than cascading the delete. `feature_id` and `workspace_id`
   both `ON DELETE CASCADE`.

2. **Two-layer Gate: SQL enforces, Go specifies.** Milestone-gating is enforced atomically
   in `ClaimAgentTask` (a new `NOT EXISTS` branch: an Issue's Milestone is unclaimable while
   any earlier Milestone — strictly lower `position`, same feature — is not `passed`). The
   existing per-agent / Initiative-status / dependency / branch gates are untouched and sit
   beside it.

   The PRD/issue also call the Gate a *pure deep module returning claimable-or-reason* (the
   "which reason" the SQL can't express — a no-row result has no reason). So
   `server/internal/gate` is the canonical pure spec: `Claimable(World) Reason` over resolved
   facts (agent-busy → initiative-inactive → milestone-gated → dependency-unmet → branch-held
   precedence) plus `MilestoneGateOpen(milestones, targetID)`, the milestone-ordering
   predicate the SQL branch mirrors. It is TDD'd (18 table-driven cases, failing-first → green)
   and is the foundation the Orchestrator (issue 10) consumes. This mirrors issue 06's posture
   (pure `internal/initiative` status module alongside the SQL status gate). The umbrella lock
   is intentionally NOT modeled here: it's a dispatch-time park (`WaitTaskForLocalDirectory`),
   not part of the claim predicate.

3. **Demo path (DB test).** `claim_milestone_gate_test.go` reuses the `initiativeGateFixture`
   builders: a two-Milestone `running` Initiative — the first Milestone's Issue claims
   immediately (no earlier sibling), the second's stays unclaimable until the first is set
   `passed`, then claims. Mirrors the established `claim_*_gate_test.go` pattern.

4. **Minimal board display.** Added `milestone_id` to the `ListIssues` SQL/handler/row scan
   and to `IssueResponse`/`IssueSchema` so the board's status view carries it; a read-only
   `GET /api/milestones` endpoint + `api.listMilestones()` + `milestoneListOptions` mirror the
   feature-chip data path. The board card renders a milestone-title chip beside the feature
   chip, gated on the same `feature` card-property toggle. Grouped/assignee and open-issue
   paths don't carry `milestone_id` yet (the schema field is optional so they don't fail
   validation) — the richer Initiative/Milestone UI is issues 16/17. Creation stays the MCP's
   job (issue 14), so the endpoint is list-only for now.

### Files changed

- **New:** `server/migrations/007_milestone.{up,down}.sql`; `server/pkg/db/queries/milestone.sql`
  (regenerated `generated/milestone.sql.go`); `server/internal/gate/{gate.go,gate_test.go}`;
  `server/internal/handler/{milestone.go,claim_milestone_gate_test.go}`;
  `packages/core/types/milestone.ts`; `packages/core/milestones/queries.ts`.
- **Gate SQL:** `server/pkg/db/queries/agent.sql` (`ClaimAgentTask` milestone branch);
  regenerated `agent.sql.go`, `models.go` (Milestone struct).
- **Issue wire `milestone_id`:** `server/pkg/db/queries/issue.sql` (`ListIssues` SELECT) +
  regenerated `issue.sql.go`; `server/internal/handler/issue.go` (raw ListIssues SELECT/scan,
  `IssueResponse`, `issueToResponse`, `issueListRowToResponse`); `packages/core/api/schemas.ts`
  (`IssueSchema.milestone_id`).
- **Milestone API/UI:** `server/cmd/server/router.go` (`/api/milestones`);
  `packages/core/api/{client.ts,schemas.ts,schemas.test.ts}` (Milestone schema + list method +
  malformed-response tests); `packages/core/{types/index.ts,package.json}` (export +
  `./milestones/queries` subpath); `packages/views/issues/components/board-card.tsx`
  (milestone chip).

### Verification

- `go test ./internal/gate/ ./internal/initiative/ ./internal/handler/` → 757 passed (incl. the
  new gate + milestone-gate tests). `go test ./internal/service/ ./cmd/server/` → only
  `TestWebSocketIntegration` fails — the documented pre-existing WS auth-handshake TCP flake,
  unrelated. `go build ./...` + `go vet ./...` clean.
- `pnpm typecheck` green; `pnpm test` green (677 views + core, incl. new milestone schema tests).
- `pnpm --filter @multica/core lint` clean (one pre-existing unrelated warning);
  `@multica/views` lint shows only 4 pre-existing errors (the `waiting` literal in WaitingBlock
  and `via .multica` in AmbientProjectBar, both from prior commits) — the milestone chip renders
  a dynamic value and adds no new lint finding.

### Notes for the next iteration

- The pure `gate` module is the spec; its runtime consumer is the Orchestrator (issue 10). When
  10 lands, wire `gate.Claimable` into the dispatch-reasoning path so the "which reason" is
  surfaced (e.g. the feature view's ready/blocked split could fold in `milestone_gated`).
- Milestone CREATE/UPDATE lands with the MCP control plane (issue 14); the validator Run that
  flips `validation_status` to `passed`/`failed` lands in issue 09 (`SetMilestoneValidationStatus`
  is already provided).
- Grouped/assignee board and open-issue serializers still omit `milestone_id`; add it there if
  issue 16's Initiative view needs milestone chips outside the status board.
