# Issue 01: Validate `branch_slug` at the MCP and HTTP boundaries

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/initiative-runner-autonomy/PRD.md` — Module #4.

## What to build

A `branch_slug` is concatenated into the branch name as `feature/<slug>` by `feature.Resolve`. Today nothing validates it, so a slug of `feat/x` silently becomes `feature/feat/x` and the worker pushes to a branch the human never intended.

Add a pure validator in the `feature` package — `ValidateBranchSlug(slug string) error` — and enforce it everywhere a `branch_slug` enters the system: the MCP `create_initiative` and `update_initiative` tools, and the corresponding feature HTTP handler. Validation happens before the value is stored; `feature.Resolve` stays a pure name-builder and is NOT the enforcement point.

Rules:
- Empty slug is valid — it means "no override" (`Resolve` falls back to the identifier). Clearing the slug via an empty string on update remains allowed.
- Reject a slug containing the `feature/` prefix (the system adds it).
- Reject a slug containing a path separator (`/`).
- Reject a slug containing characters that are not valid in a git ref name.
- On rejection, return an actionable error naming the offending condition, so the human can fix it on the first try.

## Acceptance criteria

- [ ] `feature.ValidateBranchSlug` exists as a pure function and rejects `feature/x`, `feat/x`, any slash, and invalid git-ref characters; accepts valid slugs (e.g. `todo-v3`, `auth`) and empty string.
- [ ] `create_initiative` (MCP) rejects an invalid `branch_slug` with an actionable error and does not create the Initiative.
- [ ] `update_initiative` (MCP) applies the same validation; passing empty string still clears the slug.
- [ ] The feature HTTP handler enforces the same validation on create and update.
- [ ] Table-driven tests cover accept/reject cases with the rejection reason; a boundary test confirms the MCP tools surface the error. Prior art: `branch_test.go`, `target_test.go`, `tools_feature_test.go`.
- [ ] `make check` passes (known Windows `repocache` git-clone flakes excepted).

## Blocked by

None - can start immediately.

## Comments

### Key decisions

- `ValidateBranchSlug` lives in `server/internal/feature/validate.go` alongside the existing `branch.go`. It is a pure function with no I/O.
- Slug containing `"feature/"` is rejected first (most specific), then any `/`, then git-ref sequences (`..`, `@{`), then leading/trailing dot, then `.lock` suffix, then individual invalid characters.
- Empty string passes — it's the "no override" sentinel and `Resolve` handles the fallback.
- The MCP handlers validate before building the request body, so the backend is never called with an invalid slug.
- The HTTP handler validates after body parsing, before any DB work.
- `update_feature` allows empty string (clears the field); validation only runs on non-empty slugs via `ValidateBranchSlug`'s own empty-returns-nil guard.

### Files changed

- `server/internal/feature/validate.go` — new pure validator
- `server/internal/feature/validate_test.go` — 28 table-driven cases (accept + reject with reason check)
- `server/internal/mcp/tools_feature.go` — validation in `handleCreateFeature` and `handleUpdateFeature`
- `server/internal/mcp/tools_feature_test.go` — 7 new boundary tests
- `server/internal/handler/feature.go` — validation in `CreateFeature` and `UpdateFeature`

### Test results

902 tests pass across `internal/feature`, `internal/mcp`, `internal/handler`. Pre-existing environment failures in `pkg/agent` (missing `opencode`/`kiro`/`kimi`/`hermes` binaries) and `pkg/redact` (Windows path test) are unrelated to this change.
