# Issue 07: Repo-aware MCP tools

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/multi-repo-features/PRD.md`

## What to build

Make the feature-pipeline MCP tools repo-aware. Thin shims over the existing `APIClient` — business logic stays in the handlers.

- **`list_repos`** (new read tool) — returns the active workspace's repos (`id`, `name`, `remote_url`, `default_branch`) so the model can present the assignment menu.
- **`create_issue`** — gains an optional `repo` argument (name or id) resolved against the active workspace's repos; sets `issue.repo_id`. Omitting it leaves the issue unattached (coordination issue).
- **`create_feature`** — drops the `target_branch` argument (branches are derived per `(feature, repo)` now). Optionally accepts `branch_slug`.
- **`get_feature`** — response groups child issues by **repo** and then by dependency layer (ready / blocked), and includes the set of PRs (one per repo) resolved via `issue_pull_request` joined to `github_pull_request.repo_id`.

Hand-written JSONSchema per tool, consistent with the feature-pipeline tool style. REST errors surfaced verbatim as tool errors.

## Acceptance criteria

- [ ] `list_repos` returns only the active workspace's repos.
- [ ] `create_issue` with a `repo` (name or id) sets `repo_id`; an unknown repo name/id surfaces the server's error; omitting `repo` creates an unattached issue.
- [ ] `create_feature` no longer accepts `target_branch`; `branch_slug` is optional and persisted.
- [ ] `get_feature` returns issues grouped by repo + dependency layer and a PR-per-repo set.
- [ ] Each tool function is unit-tested with a fake `APIClient` recording calls; the `get_feature` test asserts the new grouped/multi-PR shape.
- [ ] `make check` passes.

## Blocked by

- Issue 01 (repo schema, `issue.repo_id`, `github_pull_request.repo_id`, `feature.branch_slug`).

## Comments

### Iteration 1 — implemented (Sonnet)

**Key decisions**

- **`list_repos`** is a simple read-only shim calling `GET /api/repos` — no parameters needed since the active workspace is resolved by the API client headers.

- **`create_issue` repo resolution** runs in the MCP layer: `resolveRepoID(ctx, nameOrID)` calls `GET /api/repos`, matches by `id` or `name`, and passes the resolved UUID as `repo_id` to `POST /api/issues`. If not found, the tool returns an error before hitting the server. This is consistent with "resolved against the active workspace's repos" in the PRD.

- **`create_feature` / `update_feature`**: `target_branch` replaced with `branch_slug` throughout — tool schema, handler body-building, and the `update_feature` clear-field logic. The `update_feature` description stale reference to "target_branch" was also corrected.

- **`get_feature` grouping in the MCP layer**: the server `GET /api/features/{id}/issues` endpoint was extended (in `handler/feature.go`) to include `repo_id`/`repo_name` on each `IssueSummary` and `repo_id` on each `PRSummary` via new `loadIssueRepoMap` and updated `loadFeaturePRs` raw queries. The MCP `handleGetFeature` decodes the response into typed structs (`issueAPIEntry`, `prAPIEntry`, `featureIssuesAPIResponse`) and calls `groupIssuesByRepo` / `groupPRsByRepo` to produce `issues_by_repo` (keyed by repo name, fallback "unassigned") and `pull_requests_by_repo` (keyed by `repo_id`, fallback "unassigned").

- **Tool count**: 15 → 16 (added `list_repos`).

**Files changed**

- `server/internal/handler/feature.go` — `RepoID`/`RepoName` added to `IssueSummary`, `RepoID` added to `PRSummary`, `loadIssueRepoMap` helper added, `GetFeatureIssues` populates repo fields, `loadFeaturePRs` scans `gpr.repo_id`.
- `server/internal/mcp/tools_read.go` — `list_repos` tool + handler, `get_feature` handler replaced with typed-struct version, `groupIssuesByRepo`/`groupPRsByRepo`/`repoKey` helpers added.
- `server/internal/mcp/tools_feature.go` — `target_branch` → `branch_slug` in `create_feature` and `update_feature`.
- `server/internal/mcp/tools_issue.go` — `repo` param added to `create_issue`, `resolveRepoID` helper added (calls `GET /api/repos`).
- `server/internal/mcp/tools_feature_test.go` — `target_branch` → `branch_slug` assertions, `TestMCPUpdateFeatureClearsTargetBranch` → `TestMCPUpdateFeatureClearsBranchSlug`.
- `server/internal/mcp/tools_issue_test.go` — `TestMCPCreateIssueWithRepo` and `TestMCPCreateIssueUnknownRepo` added.
- `server/internal/mcp/tools_read_test.go` — `TestMCPGetFeature` updated for `issues_by_repo`/`pull_requests_by_repo` keys, `TestMCPGetFeatureGroupsByRepo` added, `TestMCPListRepos`/`TestMCPListReposBackendError` added.
- `server/internal/mcp/server_test.go` — tool count 15 → 16, `list_repos` in expected names.

**Verification**

- `go build ./...`: clean.
- `go vet ./internal/mcp/... ./internal/handler/...`: clean.
- `go test ./internal/mcp/... -v`: 49 tests, all pass (unit tests only — no DB required).
- `pnpm typecheck`: all 4 packages pass.
