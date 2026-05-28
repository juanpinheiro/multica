# Issue 10: MCP issue write tools — create, update, status, assign, comment, link_dependency

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/feature-pipeline/PRD.md`

## What to build

Six MCP tools that let Claude Code create and manage issues under features.

**Tools** (in `server/internal/mcp/tools_issue.go`):

| Tool | Input | Calls | Returns |
|---|---|---|---|
| `create_issue` | `feature_id: string` (required), `title: string`, `description: string`, `acceptance_criteria?: string[]`, `priority?: string`, `assignee_id?: string`, `assignee_type?: 'agent' \| 'member'` | `POST /api/issues` | Created issue with identifier |
| `update_issue` | `issue_id: string`, plus any subset of `title`, `description`, `acceptance_criteria`, `priority` | `PATCH /api/issues/{id}` | Updated issue |
| `set_issue_status` | `issue_id: string`, `status: string` (valid issue status values) | `PATCH /api/issues/{id}` | Updated issue |
| `assign_issue` | `issue_id: string`, `assignee_id: string`, `assignee_type: 'agent' \| 'member'` | `PATCH /api/issues/{id}` | Updated issue |
| `comment_on_issue` | `issue_id: string`, `body: string` | `POST /api/issues/{id}/comments` | Created comment |
| `link_issue_dependency` | `issue_id: string`, `depends_on_issue_id: string`, `type: 'blocks' \| 'related'` | `POST /api/issues/{id}/dependencies` (create endpoint if not present) | Created dependency row |

**`create_issue` requires `feature_id`** — this enforces the rule "every issue belongs to a feature (PRD)". A model trying to create an orphan issue gets a JSONSchema validation error before the call even goes out.

**`link_issue_dependency` may require a new backend endpoint.** Check `server/internal/handler/issue*.go` first — if no dependency-link endpoint exists, ship a minimal one (`POST /api/issues/{id}/dependencies` accepting `{depends_on_issue_id, type}`) as part of this issue. The CHECK constraint on `issue_dependency.type` already restricts to `blocks | blocked_by | related`; the endpoint can accept `blocks` and `related` only, hiding `blocked_by` as redundant inverse (rendered on the dependent issue's side).

**Tests**: unit tests with fake `APIClient`. Assert HTTP method, path, body. Assert `create_issue` rejects missing `feature_id`. If the dependency endpoint is new, integration test for it too (the issue_dependency row gets created, returned correctly, idempotency on duplicate links).

## Acceptance criteria

- [ ] All six tools registered and discoverable.
- [ ] `create_issue` requires `feature_id` (JSONSchema enforces this).
- [ ] `link_issue_dependency` creates a row in `issue_dependency` table; backend endpoint exists or is added by this issue.
- [ ] Dependency type accepted: `blocks`, `related`. `blocked_by` is not exposed as an input.
- [ ] Backend validation errors (invalid assignee, invalid status, duplicate dependency) are surfaced as tool errors.
- [ ] Unit tests per tool.
- [ ] End-to-end smoke: from Claude Code, create a feature → create three issues under it → link two of them with a `blocks` dependency → confirm via `get_feature` that the dependency is reflected.

## Blocked by

- `.scratch/feature-pipeline/issues/01-rename-project-to-feature.md`
- `.scratch/feature-pipeline/issues/07-mcp-server-skeleton.md`

## Comments

### Key decisions made

1. **New backend endpoint `POST /api/issues/{id}/dependencies`** — No existing HTTP endpoint handled issue dependencies. Added `server/internal/handler/issue_dependency.go` with `CreateIssueDependency` handler using raw SQL via `h.DB` (same pattern as `GetFeatureIssues`).

2. **`blocked_by` hidden at the HTTP layer** — The `issue_dependency` DB constraint allows `blocks | blocked_by | related`. The endpoint only accepts `blocks` and `related` from callers; `blocked_by` is the redundant inverse and is not exposed as an input type.

3. **Idempotency via SELECT-before-INSERT** — No UNIQUE constraint exists on `issue_dependency`, so duplicates are prevented by checking for an existing row first. A second POST with identical `(issue_id, depends_on_issue_id, type)` returns the existing row with HTTP 200 instead of inserting again.

4. **PATCH for issue update tools** — `update_issue`, `set_issue_status`, and `assign_issue` all use `PatchJSON` (PATCH HTTP method), consistent with the existing `tools_feature.go` pattern and the spec's stated method. A `r.Patch("/", h.UpdateIssue)` route was added to the router alongside the existing `r.Put` so the real server accepts PATCH calls. `UpdateIssue` already implements partial-update semantics via rawFields detection.

5. **`comment_on_issue` sends `content` not `body`** — The MCP tool parameter is named `body` (matching the spec and user-facing convention), but the REST endpoint expects `content` in the JSON body (`CreateCommentRequest.Content`). The translation happens transparently in `handleCommentOnIssue`.

6. **Tool count updated 9 → 15** — `server_test.go`'s `TestMCPServerToolsList` asserts all 15 tool names.

### Files changed

- `server/internal/mcp/tools_issue.go` — new file: 6 tools and handlers
- `server/internal/mcp/tools_issue_test.go` — new file: 19 unit tests covering all six tools
- `server/internal/handler/issue_dependency.go` — new file: `CreateIssueDependency` handler + request/response types
- `server/internal/handler/issue_dependency_test.go` — new file: 4 integration tests (create, idempotency, invalid type, wrong workspace)
- `server/cmd/server/router.go` — added `r.Patch("/", h.UpdateIssue)` + `r.Post("/dependencies", h.CreateIssueDependency)` to issues subrouter
- `server/internal/mcp/server.go` — added `registerIssueTools(srv)` call in `New()`
- `server/internal/mcp/server_test.go` — updated tool count assertion 9 → 15

### Blockers or notes for next iteration

- End-to-end smoke test (acceptance criterion 7) is not automated — requires a running backend and a live Claude Code session with the MCP server configured. All other acceptance criteria are satisfied with automated tests.
- Pre-existing Windows-specific failures in `repocache/*` and `execenv` tests are unchanged (documented in Issues 04 and 06).
