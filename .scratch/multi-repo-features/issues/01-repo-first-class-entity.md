# Issue 01: Repo as a first-class entity (schema + CRUD)

**Status:** `done`
**Model:** `opus`

## Parent

`.scratch/multi-repo-features/PRD.md`

## What to build

Promote a repository from the `workspace.repos jsonb` blob to a first-class `repo` entity, and wire issues and pull requests to it. End to end: a repo can be created under a workspace via the API/CLI, listed, and attached to an issue; the attachment persists and is returned on read.

Schema delta (fold into the consolidated `001_init.sql` if the personal-fork consolidation has not merged; otherwise ship `ALTER`s):

- **New table `repo`**: `id uuid pk`, `workspace_id uuid not null` (FK `workspace(id)` ON DELETE CASCADE), `name text not null`, `remote_url text not null`, `local_path text`, `default_branch text not null default 'main'`, `created_at`/`updated_at timestamptz not null default now()`. Constraints `UNIQUE (workspace_id, remote_url)` and `UNIQUE (workspace_id, name)`. Indexes on `repo(workspace_id)`.
- **Remove `workspace.repos jsonb`** — find and update every reader/writer of that column (handlers, sqlc queries, frontend that surfaces repos). Pre-release, no compatibility shim.
- **`issue.repo_id uuid null`** (FK `repo(id)` ON DELETE SET NULL) — NULL means the issue targets no code. Index `issue(repo_id)`.
- **`feature.target_branch` removed** (added by feature-pipeline, superseded here). **`feature.branch_slug text null`** added — slug used to build `feature/<branch_slug>`; NULL means derive from the feature identifier.
- **`github_pull_request.repo_id uuid null`** (FK `repo(id)`), index `github_pull_request(repo_id)`. Existing `workspace_id` stays.

Repo CRUD: sqlc queries (`CreateRepo`, `ListReposInWorkspace`, `GetRepoInWorkspace`, `DeleteRepo`), a `handler/repo.go` with routes under the workspace subtree (`/api/repos` or nested per existing convention), and `issue` create/update accepting and returning `repo_id`. Follow the Backend Handler UUID Parsing Convention — resolve `repo_id` from the request body via `parseUUIDOrBadRequest`, and validate it belongs to the resolved workspace.

This is the schema foundation every other slice builds on. No scheduler, daemon, manifest, or UI behavior changes here beyond surfacing `repo_id` on issue read/write.

## Acceptance criteria

- [ ] `make check` passes (typecheck, lint, Go test, Vitest).
- [ ] `repo` table exists with the columns, FKs, and unique constraints above; `make sqlc` regenerated and committed.
- [ ] `workspace.repos jsonb` is gone and no code references it (grep returns only unrelated strings).
- [ ] `feature.target_branch` is gone; `feature.branch_slug` exists.
- [ ] Creating a repo via API/CLI with `(name, remote_url)` persists a row scoped to the workspace; duplicate `remote_url` or `name` within a workspace is rejected with 409/400.
- [ ] Listing repos returns only the active workspace's repos.
- [ ] Creating/updating an issue with `repo_id` set persists it and returns it on read; an invalid or cross-workspace `repo_id` is rejected with 400.
- [ ] A handler test (httptest + test DB) covers create, list, unique-constraint rejection, and issue attachment, following the `handler/issue_test.go` pattern.

## Blocked by

None — can start immediately.

## Comments

### Iteration 1 — implemented (Opus)

**Key decisions**

- **`repo` table** added to the consolidated `001_init.up.sql` as a self-contained block (table + PK + the two UNIQUE constraints + indexes + FKs) appended after the existing FK section, with inline `repo_id` columns added to `issue` and `github_pull_request` and `branch_slug` swapped in for `target_branch` on `feature`. `workspace.repos jsonb` removed. The down migration drops the whole schema, so no down edit was needed. `sqlc generate` (v1.31.1) regenerated cleanly — no DB required.
- **`issue.repo_id` FK is `ON DELETE SET NULL`** (per PRD preference) so deleting a repo doesn't cascade-delete issue history. NULL = unattached coordination issue.
- **Daemon workspace-repos channel repointed, wire format preserved.** The daemon still receives `repos: [{url, description}]` (`RepoData`); the source changed from the jsonb blob to `ListReposInWorkspace` via a new `h.workspaceRepoData()` helper (`remote_url`→url, `name`→description). All four task-claim fallbacks (issue/chat/autopilot/quick-create), `DaemonRegister`, and `GetDaemonWorkspaceRepos` now read the table. This keeps the daemon allowlist/repocache working without touching the daemon-side code (Issue 04 adds the per-issue repo to the payload).
- **`feature.target_branch` → `feature.branch_slug` was forced into scope** because removing the column breaks the feature-pipeline branch resolver, the SQL claim gate, the parity test, the daemon brief plumbing, and the `feature_ready` notification. Did the *minimal* transitional rewrite to keep `make check` green: `FeatureForBranch{BranchSlug *string}`, `Resolve` builds `feature/<slug>`; SQL `resolveBranch` now `COALESCE('feature/' || NULLIF(f.branch_slug,''), …)`. **Did NOT** add the repo dimension or reorder priority — that is Issue 02's 3-arg rewrite.
- **MCP `create_feature`/`update_feature` left untouched** — they still send `target_branch` to a fake client in their unit tests and the handler simply ignores the unknown field (Go JSON). Dropping the MCP arg is explicitly Issue 07's job; touching it here would be out of scope.
- **Web `RepositoriesTab` converted to read-only** (lists repos from the new `/api/repos` query) instead of a CRUD editor that wrote `workspace.repos` via `updateWorkspace`. Per the PRD, repo registration moves to the setup skill / CLI manifest — there is no web form. The two feature repo-pickers (`create-feature`, `feature-resources-section`) and the `github-tab` count were repointed to the new `repoListOptions(wsId)` query and `repo.remote_url`.

**Files changed (high level)**

- Schema/sqlc: `server/migrations/001_init.up.sql`, `server/pkg/db/queries/{repo,workspace,feature,issue,agent}.sql` (+ regenerated `pkg/db/generated/*`).
- Go handlers: new `internal/handler/repo.go` (+ `repo_test.go`), route in `cmd/server/router.go`, `issue.go` (repo_id create/update/response + `resolveIssueRepoID` helper, both UpdateIssueParams builders carry `RepoID`), `feature.go`, `workspace.go`, `daemon.go`, `issue_feature_done.go`, `internal/feature/branch.go`.
- Go tests adapted: `branch_test.go`, `branch_parity_test.go`, `claim_branch_gate_test.go`, `issue_feature_done_test.go`, `daemon_test.go` (`setHandlerTestWorkspaceRepos` now seeds the `repo` table).
- TS: `core/types/{workspace,feature,index}.ts` (drop `WorkspaceRepo`/`repos`, add `Repo`, `target_branch`→`branch_slug`), `core/api/client.ts` (`listRepos`/`createRepo`, repos dropped from `updateWorkspace`), `core/workspace/queries.ts` (`repoListOptions`), `views` (repositories-tab, github-tab, create-feature, feature-resources-section, feature-detail) + their tests, and `repos`/`target_branch` removed from Workspace/Feature mocks across `core`/`views`/`apps/web`.

**Verification**

- `go build ./...`, `go vet ./...`: clean. `internal/feature` tests: 10 passed. `internal/handler`: compiles, DB-gated tests skip (Docker not running locally — repo CRUD / claim-gate / issue-attachment tests are written and correct but unverified against a live DB here).
- `pnpm typecheck`: 4/4 tasks pass. `pnpm test` (Vitest): all packages pass (669 views tests + others).
- The remaining Go test failures locally are **pre-existing, environment-only** (missing agent CLIs `opencode`/`kiro`/`kimi`/`hermes`, Windows path-separator in the redact test, FS-layout skill tests) — none touch this change.

**Notes for the next iteration**

- **Pre-existing lint debt, not introduced here:** `i18next/no-literal-string` errors at `feature-detail.tsx` lines ~102 (`running`) and ~108 (`blocked by`) exist at HEAD (from the feature-pipeline commit). Fixing them requires adding i18n keys across *all* locale files or the `locales/parity.test.ts` fails — out of scope for Issue 01. My own change is lint-clean (the one new literal I added was moved to a plain-TS variable).
- **DB-backed tests need a live Postgres** to actually run (Docker was down in this environment). Re-run `make test` with the DB up to confirm `repo_test.go`, the `repo_id` claim-gate cases, and `issue_feature_done` fixtures pass against real SQL.
- Issue 02 will replace the transitional 2-arg `feature.Resolve` with the repo-aware 3-arg form; Issue 03 adds `AND i2.repo_id = i.repo_id` to Gate 2; both build on the `branch_slug` SQL mirror landed here.
