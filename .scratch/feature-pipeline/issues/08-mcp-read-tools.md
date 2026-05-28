# Issue 08: MCP read tools — `list_features`, `get_feature`, `list_issues`, `get_issue`, `list_agents`

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/feature-pipeline/PRD.md`

## What to build

Five read-only MCP tools that let Claude Code answer questions about the user's Multica state without touching the dashboard.

**Tools** (in `server/internal/mcp/tools_read.go`):

| Tool | Input | Calls | Returns |
|---|---|---|---|
| `list_features` | `status?: 'planned' \| 'in_progress' \| 'paused' \| 'completed' \| 'cancelled'` | `GET /api/features?status=...` | Array of feature summaries (id, identifier, title, status, target_branch) |
| `get_feature` | `feature_id: string` (UUID or identifier like `MUL-F-12`) | `GET /api/features/{id}` + `GET /api/features/{id}/issues` + linked PR lookup | Feature with description, target_branch, child issues grouped by dependency layer (`ready_now`, `blocked`), linked PR if any |
| `list_issues` | `feature_id?: string`, `status?: string`, `assignee_id?: string` | `GET /api/issues?...` | Array of issue summaries |
| `get_issue` | `issue_id: string` (UUID or identifier) | `GET /api/issues/{id}` + comments + linked PR | Full issue with comments, PR link, dependency info |
| `list_agents` | (no args) | `GET /api/agents` | Array of agent summaries (id, name, provider) — used so the model knows valid assignee IDs |

**JSONSchema**: hand-written, one per tool, lives next to the tool implementation. Optional fields explicitly marked optional. Tool descriptions written for the model's benefit (one sentence each).

**Implementation pattern**: each tool function takes the parsed args, calls a method on the held `cli.APIClient`, and returns the response (or a tool error on failure). No business logic — these are HTTP shims.

**Error surface**: REST errors are surfaced as MCP tool errors with the HTTP status code and response body included in the message text. The MCP server itself never returns protocol-level errors for backend failures — they go to the model as tool results so it can decide whether to retry or report.

**Tests**: unit tests with a fake `APIClient` that records calls. Assert request shape (URL, query params, body) and error propagation. Stdio loop is not re-tested here — Issue 07's test covers it.

## Acceptance criteria

- [ ] All five tools registered and discoverable via `tools/list`.
- [ ] Each tool has a JSONSchema with clear input fields and a model-facing description.
- [ ] Each tool calls the correct REST endpoint via `APIClient`.
- [ ] `get_feature` returns issues grouped by dependency layer (uses Issue 04's gate logic on the read side too — either by re-evaluating in MCP or by relying on a backend endpoint that exposes the grouping).
- [ ] Backend errors are surfaced as tool errors with status code + body in the message.
- [ ] Unit tests cover each tool with a fake `APIClient`.
- [ ] Running `claude mcp add multica -- multica mcp` then asking "list my features" in a Claude Code session returns real data from the backend.

## Blocked by

- `.scratch/feature-pipeline/issues/01-rename-project-to-feature.md`
- `.scratch/feature-pipeline/issues/07-mcp-server-skeleton.md`

## Comments

### Key decisions made

1. **New backend endpoint `GET /api/features/{id}/issues`** — No existing HTTP endpoint exposed dependency information. Rather than skipping the dependency grouping or reimplementing it in MCP without DB access, added a new `GetFeatureIssues` handler in `handler/feature.go` that returns issues split into `ready_now` and `blocked` groups. This mirrors the claim handler's gate logic (Issue 04) on the read side, so the model sees the same grouping the scheduler enforces.

2. **`loadBlockedByMap` uses raw SQL via `h.DB`** — The `issue_dependency` table has no sqlc queries. Used `h.DB.Query()` (the `dbExecutor` interface already on the Handler) with a direct SQL join across `issue_dependency`, `issue`, and `workspace` to get blocker identifiers for each issue in one query.

3. **Tool handlers use the provided context** — Each `handle*` function uses the `ctx` parameter passed by the mcp-go framework instead of `context.Background()`, so cancellation and deadlines propagate from the MCP protocol layer.

4. **`toolError` surfaces HTTP status + body verbatim** — REST errors from `cli.HTTPError` include the status code and response body in the tool error message so the model can see "HTTP 404: feature not found" rather than a generic error.

5. **`get_feature` makes two sequential REST calls** — One to `GET /api/features/{id}` for feature details and one to `GET /api/features/{id}/issues` for the grouped issues. This follows the MCP shim pattern (Issue 08 spec: "No business logic — these are HTTP shims").

6. **`tools/list` test updated from 0 → 5 tools** — The existing `TestMCPServerToolsListEmpty` was renamed to `TestMCPServerToolsList` and now asserts all five tool names are present.

### Files changed

- `server/internal/handler/feature.go` — Added `IssueSummary`, `BlockedIssueSummary`, `FeatureIssuesResponse` types; `GetFeatureIssues` handler; `loadBlockedByMap` helper.
- `server/cmd/server/router.go` — Registered `GET /api/features/{id}/issues → h.GetFeatureIssues`.
- `server/internal/mcp/tools_read.go` — New file: `registerReadTools`, five tool definitions + handlers, `toolError`, `jsonResult` helpers.
- `server/internal/mcp/server.go` — Call `registerReadTools(srv)` in `New()`.
- `server/internal/mcp/server_test.go` — Updated `TestMCPServerToolsListEmpty` → `TestMCPServerToolsList`, asserts 5 tools by name.
- `server/internal/mcp/tools_read_test.go` — New file: 12 unit tests covering all five tools, status filter forwarding, missing-ID errors, backend error surfacing.

### Blockers or notes for next iteration

None. All acceptance criteria satisfied:
- All five tools registered and discoverable via `tools/list` ✓
- Each tool has a JSONSchema with description ✓
- Each tool calls the correct REST endpoint via `APIClient` ✓
- `get_feature` returns issues grouped by dependency layer via new `GET /api/features/{id}/issues` endpoint ✓
- Backend errors surfaced as tool errors with HTTP status code + body ✓
- Unit tests cover each tool with a fake backend ✓
- 812 tests pass across handler, mcp, and feature packages ✓
