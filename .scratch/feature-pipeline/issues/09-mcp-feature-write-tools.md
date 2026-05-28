# Issue 09: MCP feature write tools — `create_feature`, `update_feature`, `approve_feature`, `set_feature_status`

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/feature-pipeline/PRD.md`

## What to build

Four MCP tools that let Claude Code create and manage features (PRDs).

**Tools** (in `server/internal/mcp/tools_feature.go`):

| Tool | Input | Calls | Returns |
|---|---|---|---|
| `create_feature` | `title: string`, `description: string` (markdown body of the PRD), `priority?: string`, `target_branch?: string`, `lead_id?: string` | `POST /api/features` with `status: 'planned'` (always) | Created feature with its identifier (e.g. `MUL-F-12`) |
| `update_feature` | `feature_id: string`, plus any subset of `title`, `description`, `priority`, `target_branch`, `lead_id` | `PATCH /api/features/{id}` | Updated feature |
| `approve_feature` | `feature_id: string` | `PATCH /api/features/{id}` with `status: 'in_progress'` | Updated feature |
| `set_feature_status` | `feature_id: string`, `status: 'planned' \| 'in_progress' \| 'paused' \| 'completed' \| 'cancelled'` | `PATCH /api/features/{id}` | Updated feature |

`approve_feature` is sugar for `set_feature_status({status: 'in_progress'})` because the verb is the user's mental model — the model should be able to call it without remembering enum values.

`create_feature` always starts in `planned` regardless of what the model passes; the model approves via a separate call. This makes the approval ritual explicit even when the model is automating both steps.

**Behavior contract**: every write tool surfaces backend validation errors verbatim (e.g. "target_branch may not contain whitespace") so the model can correct and retry.

**Tests**: unit tests per tool with a fake `APIClient`. Assert the right HTTP method, path, and body. Assert that `create_feature` forces `status: 'planned'` even if the model passes something else.

## Acceptance criteria

- [x] All four tools registered and discoverable.
- [x] `create_feature` accepts `target_branch` (optional) and persists it. Resulting feature has `status: 'planned'`.
- [x] `update_feature` allows partial updates (only fields the model passes are changed).
- [x] `approve_feature` flips status to `in_progress` and returns the updated feature.
- [x] `set_feature_status` accepts any valid enum value and rejects invalid ones via backend validation (surfaced as a tool error).
- [x] Unit tests cover each tool with a fake `APIClient`.
- [ ] End-to-end smoke: from Claude Code, `create_feature(...)` then `approve_feature(<id>)` creates a feature in `in_progress` status in the real backend.

## Blocked by

- `.scratch/feature-pipeline/issues/01-rename-project-to-feature.md`
- `.scratch/feature-pipeline/issues/07-mcp-server-skeleton.md`

## Comments

### Key decisions made

1. **`PatchJSON` updated to return `*HTTPError`** — was using `fmt.Errorf` unlike `PostJSON`. Updated to return `&HTTPError{...}` so `toolError` can surface the HTTP status code and body in the structured format ("HTTP X: body"). This makes PATCH error messages consistent with POST errors.

2. **`update_feature` uses `req.GetArguments()` for target_branch detection** — `target_branch` must support empty string as a valid value (to clear the field). The other fields skip empty strings because empty string = "not specified". Using `GetArguments()` to check key presence lets the handler distinguish "not passed" from "passed as empty string" for target_branch specifically.

3. **`update_feature` loop over text fields** — title, description, priority, lead_id share the same "include if non-empty" logic, extracted into a range loop over a string slice rather than four identical if-blocks.

4. **Tools registered in `registerFeatureTools()`** — mirrors the `registerReadTools()` pattern from Issue 08. Called in `New()` right after `registerReadTools()`.

5. **`server_test.go` tool count updated 5 → 9** — `TestMCPServerToolsList` now asserts all nine tool names are present.

### Files changed

- `server/internal/cli/client.go` — `PatchJSON`: changed error return from `fmt.Errorf` to `&HTTPError{...}` for HTTP 4xx/5xx responses.
- `server/internal/mcp/tools_feature.go` — new file: four tool definitions + handlers.
- `server/internal/mcp/server.go` — added `registerFeatureTools(srv)` call in `New()`.
- `server/internal/mcp/server_test.go` — updated tool count from 5 to 9, added new tool names to the assertion list.
- `server/internal/mcp/tools_feature_test.go` — new file: 14 unit tests covering all four tools (method, path, body, error cases, optional field inclusion, target_branch clearing).

### Blockers or notes for next iteration

- End-to-end smoke test (acceptance criterion 7) is not automated — requires a running backend instance and a live Claude Code session. All other acceptance criteria are satisfied with automated unit tests.
- Pre-existing Windows repocache test failures are unchanged and unrelated to this issue.
