# Issue 14: MCP control-plane surface

**Status:** `done`
**Model:** `opus`

## Parent

`.scratch/initiative-runner/PRD.md` — see ADR-0003.

## What to build

Extend the existing MCP server so the control plane (the human + Claude Code) can create and steer
Initiatives without touching the UI: tools to create/update an Initiative, its Milestones, and its Issues;
write DoD assertions; and set Initiative status (the `ready` flip and the rest of the state machine).
This is the primary creation path now that the manual UI is gone.

Model `opus`: the MCP tool surface is the control-plane API and needs a coherent, durable shape.

## Acceptance criteria

- [x] MCP tools create/update Initiative, Milestone, Issue (with `Model`) and write DoD assertions
- [x] MCP tool sets Initiative status, including the `ready` flip; illegal transitions rejected
- [x] Claude Code can create an Initiative end-to-end and flip it `ready` purely via MCP
- [x] Responses run through schema validation (no bare casts), per the API compatibility rules
- [x] `go test ./...` and `pnpm test` pass (only the pre-existing env-specific failures remain)

## Blocked by

- `06-reshape-entities-and-status-claim`

## Comments

### Key decisions

1. **Control-plane write endpoints added; the MCP tools are thin proxies over them.**
   The intelligence/validation lives in the Go handlers (single source of truth), and the MCP
   tools just shape and forward JSON — keeping the deterministic logic thin (ADR-0004 posture).
   New REST endpoints: `POST /api/milestones`, `PATCH /api/milestones/{id}`,
   `POST /api/milestones/{id}/dod`, and `PATCH /api/features/{id}` (the features route only had
   `PUT` — a latent bug, since every feature MCP tool calls `PatchJSON`; PATCH is now registered
   alongside PUT).

2. **Illegal status transitions are rejected server-side, not in the tool.** `UpdateFeature` now
   routes any status change through `initiative.Transition` via `validateInitiativeTransition`:
   a no-op (target == current) is allowed, an unknown status is a 400, an illegal move is a 422.
   The MCP `set_feature_status` / `approve_feature` tools surface the server's 4xx verbatim. This
   is the right seam — the UI mirror (issue 16) gets the same guard for free.

3. **`approve_feature` now flips `draft → ready`** (the trigger from the PRD), replacing the stale
   `in_progress` it set before the issue-06 status reshape. `set_feature_status` and `list_features`
   descriptions were updated to the new 7-state set. The tool names stay `*_feature` to match the
   physical `feature` table and the `/api/features` REST surface (issue 06 kept the physical names;
   adding parallel `*_initiative` names would fork the abstraction).

4. **Mode + budget/tolerance are settable at create and update** (ADR-0005, the wiring issue 11
   deferred here). `CreateFeature`/`UpdateFeature` SQL gained `mode, budget_tokens, budget_runs,
   budget_seconds, failure_tolerance` as `COALESCE(narg, default/col)` so omitting them keeps the
   DB defaults (hitl / no caps / tolerance 3) — existing callers and test fixtures are unaffected.
   `mode` is validated against `initiative.ValidMode` (new `initiative/mode.go`).

5. **The per-Issue `Model` preference is stored as issue metadata, not a new column.** There is no
   `model` column on `issue` (the execution-plane model comes from the assigned agent), so
   `create_issue`'s `model` param is recorded via a follow-up `PUT /metadata/model` write — a
   durable planning record that survives on the Issue without a schema change. `milestone_id` *is*
   a real column (it gates execution), so it is plumbed through `CreateIssue` proper, validated to
   belong to the same workspace.

6. **API-compatibility rule honored.** The single-item create/update responses parse through the
   existing `MilestoneSchema` / `DodAssertionSchema` (new `EMPTY_MILESTONE` / `EMPTY_DOD_ASSERTION`
   fallbacks) via `parseWithFallback` in the new TS client methods (`createMilestone`,
   `updateMilestone`, `createDodAssertion`) — no bare casts — with malformed-response tests added.

### Files changed

**Go — SQL + sqlc (regenerated `pkg/db/generated/*`)**
- `pkg/db/queries/feature.sql` — `CreateFeature`/`UpdateFeature` carry mode + budget/tolerance
- `pkg/db/queries/issue.sql` — `CreateIssue` carries `milestone_id`
- `pkg/db/queries/milestone.sql` — new `UpdateMilestone`, `CountMilestonesByFeature`

**Go — handlers / routing**
- `internal/initiative/mode.go` (new) — `Mode` type + `ValidMode`
- `internal/handler/feature.go` — create/update request fields, mode validation,
  `validateInitiativeTransition` (state-machine guard), mode/budget plumbing
- `internal/handler/milestone.go` — `CreateMilestone`, `UpdateMilestone`
- `internal/handler/dod.go` — `CreateDodAssertion`
- `internal/handler/issue.go` — `CreateIssue` accepts + validates `milestone_id`
- `internal/handler/handler.go` — `ptrToInt8` / `ptrToInt4` helpers
- `cmd/server/router.go` — `PATCH /api/features/{id}`; milestone `POST`/`{id} PATCH`/`{id}/dod POST`

**Go — MCP tools**
- `internal/mcp/server.go` — register milestone tools
- `internal/mcp/tools_feature.go` — Initiative descriptions, mode/budget params,
  `approve_feature` → ready, new status set
- `internal/mcp/tools_milestone.go` (new) — `create_milestone`, `update_milestone`,
  `create_dod_assertion`
- `internal/mcp/tools_issue.go` — `create_issue` gains `milestone_id` + `model` (metadata write)
- `internal/mcp/tools_read.go` — `list_features` status description

**Go — tests**
- `internal/handler/mcp_control_plane_test.go` (new) — 11 DB-integrated handler tests
- `internal/mcp/{server,tools_feature,tools_issue}_test.go` — updated tool count/names, ready-flip,
  mode/budget, milestone_id/model
- `internal/mcp/tools_milestone_test.go` (new) — milestone/DoD tool tests

**TypeScript**
- `packages/core/api/schemas.ts` — `EMPTY_MILESTONE`, `EMPTY_DOD_ASSERTION`
- `packages/core/api/client.ts` — `createMilestone`, `updateMilestone`, `createDodAssertion`
- `packages/core/api/schemas.test.ts` — single-item malformed-response tests

### Blockers / notes

- `pnpm typecheck`, `pnpm test` (677 views + core), and `go test` for every touched package pass.
  The only Go failure in those packages is the documented `TestWebSocketIntegration` WS-handshake
  flake; the wider `go test ./...` failures (execenv tilde, redact, daemon local-skills/symlinks,
  repocache Windows git-clone unlink, agent opencode/hermes binaries) are all pre-existing,
  environment-specific, and unrelated.
- **For issue 15 (planning skills):** `/to-prd` → `create_feature` (set `mode` + budget),
  `create_milestone`, `create_dod_assertion`; `/to-issues` → `create_issue` (set `milestone_id`
  + `model`), then `approve_feature` to flip `ready`.
- **For issue 16 (UI mirror):** the `createMilestone`/`updateMilestone`/`createDodAssertion` client
  methods and the server-side transition guard are ready; the UI status-flip can reuse the existing
  `updateFeature` client method and gets illegal-transition rejection for free.
- `model` lives in issue metadata (key `model`); a future daemon change could let it override the
  assigned agent's model at dispatch, but that is out of scope here.
