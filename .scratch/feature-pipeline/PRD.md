# PRD: Feature Pipeline — PRD-to-PR orchestration via MCP

**Status:** `ready-for-agent`
**Owner:** Juan Pinheiro
**Created:** 2026-05-27
**Depends on:** `.scratch/multica-personal-fork/PRD.md` (the stripped personal fork must land first — schema consolidation in particular)

## Problem Statement

The user's manual workflow for shipping a change today runs entirely outside Multica: a Claude Code conversation in the terminal, with the `/grill-me`, `/to-prd`, and `/to-issues` skills writing markdown files into `.scratch/<feature-slug>/`. The `.scratch/` files are read by the user, then re-read by whichever agent picks the work up. Multica — the very product the user is building — is not part of this loop. There is no place where:

- A PRD lives as structured data instead of a markdown file the user has to remember to commit.
- Issues derived from a PRD are linked back to their parent PRD with explicit dependency edges (sequential vs parallel).
- The local daemon picks up those issues automatically and runs the agent fleet against them.
- The work converges into a single PR per PRD instead of one PR per issue (which fragments review) or one giant PR per epic (which can't be reviewed at all).

The result is that the user is the human glue at every step: they invoke `/to-prd`, then they invoke `/to-issues`, then they manually open Claude Code in a new pane and tell the agent "implement issue 03", then they manually open a PR, then they manually mark the issue done in the markdown file. The dashboard (`apps/web/`) and the daemon — the two pieces of Multica that exist precisely to remove this glue — are not wired into the workflow.

Two specific friction points came up during design and need to be solved by this PRD, not papered over:

1. **Branch fragmentation.** Multica's default model — one issue → one branch → one PR — produces small PRs that are easy to review individually but lose the "feature" context. For a PRD that the user thinks of as one coherent change ("rewrite auth"), getting five disconnected PRs forces the user to either merge incomplete intermediate states into `main` or hold a chain of stacked PRs together by hand.
2. **Parallel vs sequential execution.** Some PRDs decompose into issues that can run in parallel (independent slices). Others decompose into a chain where issue 2 can only start after issue 1 produces the schema, etc. The current daemon claim loop is order-blind: it picks the oldest queued task regardless of whether its dependencies are satisfied, and it has no notion that two tasks might be fighting for the same branch.

The `issue_dependency` table exists in the schema since the initial migration (`server/migrations/001_init.up.sql:928`) but is never read by any handler, service, or query — it is structural primitive without behavior.

## Solution

Turn the existing Multica primitives — project, issue, autopilot, daemon, agent_task_queue, issue_dependency, github_pull_request — into a **feature pipeline**: a PRD enters as a `feature` row (the new name for what Multica calls "project"), gets decomposed into child issues with explicit dependency edges and an optional shared target branch, and the existing daemon fleet picks the issues up in dependency order, all converging on one PR per feature.

Concrete additions:

- **Rename `project` to `feature` across the entire fork** (schema, sqlc queries, handlers, CLI subcommand, MCP tool names, frontend routes/labels). The Multica original used "project" in the Linear sense — an iniciative umbrella above issues. In this fork's single-engineer reality, the umbrella never had multiple PRDs in it; the umbrella IS the PRD. "Feature" is the term the user, the skills, and the GitHub artifact (the feature branch) already use. Standardizing on it eliminates a vocabulary mismatch that would otherwise compound every time the MCP server names a tool or the dashboard labels a card.
- **Add `feature.target_branch`** (optional text). When set, every issue under the feature converges on this branch and the claim handler serializes their execution. When unset, each issue gets its own branch and runs in parallel as today.
- **Honor `issue_dependency` in the claim handler.** An issue with unsatisfied `blocked_by` dependencies is not eligible for dispatch. This is the gate that turns "3 sequential + 2 parallel" from a comment into actual scheduler behavior.
- **Gate the autopilot on feature status.** Issues only become eligible for dispatch when their parent feature is in status `in_progress`. This gives the user the explicit ritual "approve the PRD → motor starts running" without inventing a new approval table — it reuses `project.status` (already `planned | in_progress | paused | completed | cancelled`).
- **Ship an MCP server (`multica mcp`)** as a stdio subcommand of the existing CLI. Claude Code (and any future MCP client) gets a curated set of tools that mirror the user's workflow: create a feature, create issues, link dependencies, approve, monitor, comment, assign, mark done. The MCP server reuses the CLI's `APIClient` so authentication, profiles, and worktree isolation come for free.
- **Override the `/to-prd` and `/to-issues` skills at the project level** (`.claude/skills/`) so they call the MCP tools instead of writing markdown into `.scratch/`. `/grill-me` is unchanged — it produces conversation context that `/to-prd` then consumes.
- **Web dashboard adjustment**: the feature detail page treats `description` as primary content (the PRD body, rendered with the existing markdown component) rather than as metadata. Issues are grouped by their dependency layer ("ready now", "blocked by X"). The shared branch, if set, is shown in the header. No new pages, no new components — only rearrangement of the existing project detail view.

The result: the user opens Claude Code, runs `/grill-me` to think out loud, then `/to-prd` writes a feature into Multica via MCP, then `/to-issues` creates the issues and dependency edges via MCP, then the user opens the dashboard, reviews the spec, clicks "Approve" (or asks Claude Code to set status), and walks away. The local daemon picks the issues up in the right order, agents commit to the right branch, a single PR accumulates, the user reviews and merges.

## User Stories

1. As a solo engineer using my personal Multica, I want to run `/grill-me` in Claude Code to think out loud about a problem, so that I have raw context before committing to a spec.
2. As a solo engineer, I want to run `/to-prd` after grilling and have it create a `feature` row in Multica with the PRD as its description, so that the spec lives as structured data instead of a loose markdown file.
3. As a solo engineer, I want `/to-prd` to set the new feature's status to `planned` by default, so that nothing kicks off automatically until I explicitly approve.
4. As a solo engineer, I want `/to-prd` to optionally accept a `target_branch` argument, so that I can declare upfront "this whole feature converges on `feature/auth-v2`".
5. As a solo engineer, I want `/to-issues` to read the feature I just created and emit N implementation issues, each with `feature_id` set and acceptance criteria populated, so that the spec-to-issues link is automatic.
6. As a solo engineer, I want `/to-issues` to also create `issue_dependency` rows when slices have an obvious ordering ("API before UI"), so that the daemon knows which can run in parallel and which must wait.
7. As a solo engineer, I want issues without dependencies to be dispatched in parallel up to the daemon's concurrent slot limit, so that I don't lose throughput when slices are genuinely independent.
8. As a solo engineer, I want an issue with unsatisfied `blocked_by` dependencies to stay queued (not dispatched) until its blockers are `done`, so that the order I declared in `/to-issues` is actually enforced at runtime.
9. As a solo engineer, I want two queued issues that share the same target branch to never be dispatched concurrently, so that two agents don't push to the same branch at the same time and corrupt each other's work.
10. As a solo engineer, I want a feature whose status is not `in_progress` to have its issues stay queued without being dispatched, so that the autopilot doesn't start running before I've approved the spec.
11. As a solo engineer, I want to "approve" a feature by setting its status to `in_progress` (via MCP tool, CLI, or web button), so that approval is a single explicit action with a clear effect.
12. As a solo engineer, I want the daemon to pass the resolved target branch to each task, so that the agent CLI checks out the right branch before working.
13. As a solo engineer, I want the daemon to fetch the existing branch from `origin` if it already exists, so that the second issue's agent sees the first issue's commits and continues from there.
14. As a solo engineer, I want the task brief to include a warning when the branch is shared ("other issues of this feature also push here — don't force-push, don't rewrite history"), so that the agent doesn't accidentally destroy peer work.
15. As a solo engineer, I want a single PR to accumulate commits as each issue completes, so that I review one cohesive change set instead of five fragmented PRs.
16. As a solo engineer, I want each issue to also append a comment to the PR ("MUL-487 done: added migration for user_session table"), so that the PR description tells the story of the feature without me writing it manually.
17. As a solo engineer, I want to comment on an issue via MCP from Claude Code (`@agent please address review feedback`), so that my conversational interface to Multica doesn't require switching to the web UI.
18. As a solo engineer, I want to reassign an issue to a different agent via MCP (when claude-code is busy, fall back to codex), so that I'm not blocked by one provider's rate limit.
19. As a solo engineer, I want MCP read tools (`list_features`, `get_feature`, `list_issues`, `get_issue`) so that Claude Code can answer "what's the status of feature auth-v2?" without me opening the dashboard.
20. As a solo engineer, I want the web dashboard's feature page to show the PRD body as the primary content with a "Approve" button when status is `planned`, so that the dashboard is where I review specs, not just where I monitor tasks.
21. As a solo engineer, I want the feature page to group child issues into "ready now" and "blocked by …" sections, so that I see at a glance what the daemon will pick up next.
22. As a solo engineer, I want the feature page to display the target branch (if set) in the header with a link to the PR (if open), so that I have one click from spec to code.
23. As a solo engineer, I want the existing `multica project` CLI subcommand renamed to `multica feature`, so that the vocabulary is consistent everywhere I look (CLI, dashboard, MCP, code).
24. As a solo engineer, I want the existing project handlers, sqlc queries, frontend routes, and labels renamed from `project`/`projects` to `feature`/`features`, so that there is no place in the fork where I read "project" and have to mentally translate to "feature".
25. As a solo engineer, I want the renamed DB table and column to ship as part of the consolidated `001_init.sql` (not as a separate migration), so that the rename doesn't add migration archaeology to a freshly consolidated schema.
26. As a solo engineer, I want the MCP server distributed as a subcommand of the existing `multica` binary (`multica mcp`), so that I don't have to install or update a second tool.
27. As a solo engineer, I want the MCP server to use the existing CLI config (`~/.multica/config.toml` or `--profile`), so that authentication and workspace selection are inherited and worktree-aware automatically.
28. As a solo engineer, I want to add the MCP server to Claude Code with `claude mcp add multica -- multica mcp`, so that the setup is one command.
29. As a solo engineer, I want the MCP toolset deliberately small (under fifteen tools), so that the system prompt overhead in Claude Code stays low and the model doesn't get confused by hundreds of similar endpoints.
30. As a solo engineer, I want the MCP tools to map one-to-one to the workflow verbs (create feature, create issue, link dependency, approve, comment, assign, set status, get/list), so that the tool names read like English imperatives.
31. As a solo engineer, I want each MCP tool to call the corresponding REST endpoint through the existing `APIClient`, so that there is exactly one place where business logic lives (the handler) and the MCP server is a thin shim.
32. As a solo engineer, I want the MCP server to fail loudly when the underlying REST call returns an error (validation, conflict, not-found), surfacing the server's error message verbatim, so that I can debug from the Claude Code transcript without tailing server logs.
33. As a solo engineer, I want the MCP server to refuse to run when `MULTICA_TOKEN` is missing or the server is unreachable, with a clear instruction on how to fix it, so that the first-run experience is not a stack trace.
34. As a solo engineer, I do NOT want a web UI for creating PRDs or issues — that surface remains read-only-with-approval. All creation happens through Claude Code via MCP, so that there is exactly one path into the system and the dashboard's job is monitoring.
35. As a solo engineer, I do NOT want a remote/HTTP transport for the MCP server in v1, since stdio is enough for local Claude Code and adding HTTP requires per-client token management I don't need yet.
36. As a solo engineer, I do NOT want the MCP server to bypass the REST API and write to Postgres directly, so that there are no two ways to do the same thing and the audit trail (existing activity feed) stays accurate.
37. As a solo engineer, I do NOT want a separate "PRD" entity beyond renaming `project` to `feature`, since the existing fields (title, description, status, priority, lead, resources) already cover everything a PRD needs.
38. As a solo engineer, I do NOT want a dependency type beyond `blocks`/`blocked_by` in v1; `related` is already a valid value in the existing CHECK constraint but the claim handler treats it as non-gating.
39. As a solo engineer, I want failures of a blocking issue to leave dependents stuck in `queued` indefinitely (no silent skip, no time-out auto-promote), so that a real failure forces me to look at it instead of cascading into more failures.
40. As a solo engineer, I want the rename to ship in a single commit/PR so that `git log` for the rename is one line, since searching across `project` → `feature` references is a chore that benefits from atomic before/after.

## Implementation Decisions

### Rename `project` → `feature`

The Multica original uses "project" as the Linear-style umbrella above issues. In this fork's single-engineer workflow, the umbrella IS the PRD and there is no level above it. The whole vocabulary gets renamed to remove the mismatch.

- **Database**: table `project` → `feature`; FK column `issue.project_id` → `issue.feature_id`; table `project_resource` → `feature_resource`; FK `project_resource.project_id` → `feature_resource.feature_id`. The CHECK constraints (`feature_priority_check`, `feature_lead_type_check`, `feature_status_check`) are renamed accordingly. Existing status enum values (`planned | in_progress | paused | completed | cancelled`) are preserved verbatim — they read fine as feature lifecycle states.
- **sqlc queries**: every query name renamed (`ListProjects` → `ListFeatures`, etc.), `:many`/`:one`/`:exec` annotations unchanged.
- **Go handlers**: `handler/project.go` → `handler/feature.go`, `handler/project_resource.go` → `handler/feature_resource.go`, struct types (`ProjectResponse` → `FeatureResponse`), HTTP routes (`/api/projects/...` → `/api/features/...`), test files renamed and grep-replaced for the type/route changes.
- **CLI**: `multica project ...` subcommand and the `cmd_project.go` file renamed. The `project` subcommand alias is NOT preserved — the fork is pre-release and there are no external scripts depending on the old name.
- **Frontend**: package paths (`packages/views/projects/` → `packages/views/features/`), route segments (`/[ws]/projects/...` → `/[ws]/features/...`), navigation entries, sidebar labels, i18n strings in `locales/en/`, all references in shared Zustand stores (`use-project-store` → `use-feature-store`). The single `Project` symbol exported by `@multica/core` becomes `Feature`; downstream imports get updated mechanically.
- **MCP tools**: named with `feature` from day one (`create_feature`, `list_features`, etc.).
- **Reserved slug list**: `projects` is removed from `reserved_slugs.json`; `features` is added. The generator re-runs (`pnpm generate:reserved-slugs`).

The rename lands as a single PR with the schema change folded into the consolidated `001_init.sql` if that consolidation hasn't merged yet, or as an `ALTER TABLE project RENAME TO feature` plus column rename if it has.

### New schema additions

- `feature.target_branch text NULL` — when set, the branch name that every child issue's task will check out and push to. NULL means "issues get isolated branches (current behavior)".
- No other new columns. No new tables. `issue_dependency` is reused as-is.

### Claim handler — two new gates

The claim query in the task service (`server/internal/service/task.go::ClaimTaskForRuntime` or equivalent — exact location confirmed during implementation) gains two `NOT EXISTS` clauses:

```sql
-- Gate 1: dependencies satisfied (blocking deps only)
AND NOT EXISTS (
  SELECT 1 FROM issue_dependency d
  JOIN issue b ON d.depends_on_issue_id = b.id
  WHERE d.issue_id = i.id
    AND d.type IN ('blocks','blocked_by')
    AND b.status != 'done'
)

-- Gate 2: target branch not currently held by another dispatched task
AND NOT EXISTS (
  SELECT 1 FROM agent_task_queue t2
  JOIN issue i2 ON t2.issue_id = i2.id
  LEFT JOIN feature f2 ON i2.feature_id = f2.id
  WHERE t2.status = 'dispatched'
    AND t2.id != t.id
    AND resolveBranch(i2, f2) = resolveBranch(i, f)
)

-- Existing: feature must be approved before issues become eligible
AND (f.status IS NULL OR f.status = 'in_progress')
```

The third clause (feature status gate) makes issues created under an unapproved feature stay queued. Issues with no feature (`feature_id IS NULL`) are exempt — they behave as today.

`resolveBranch(i, f)` is implemented in SQL via `COALESCE(f.target_branch, i.metadata->>'target_branch', 'issue/' || i.identifier)`.

### Deep module: branch resolver

`server/internal/feature/branch.go` (new package) — a pure function with this signature:

```go
type Feature struct { TargetBranch *string }
type Issue   struct { Identifier string; Metadata map[string]any }

func Resolve(i Issue, f *Feature) (branch string, shared bool)
```

Pure, no DB, no I/O. Called from two places: the SQL `resolveBranch` mirror (kept consistent by a test that asserts SQL and Go produce identical results for a fixture set) and the claim handler when populating the outbound `Task` payload's `TargetBranch` and `IsSharedBranch` fields.

This module is the test target par excellence: small surface, total coverage feasible, all the interesting cases (NULL feature, NULL feature.target_branch, NULL metadata, override on issue, both set, etc.) are pure data.

### Task payload extension

`server/internal/daemon/types.go::Task` gains:

```go
TargetBranch   string // resolved branch name (always set)
IsSharedBranch bool   // true when branch came from feature.target_branch
```

Populated by the claim handler after the SQL select returns, using the `feature.Resolve` function. Existing fields untouched.

### Daemon: branch checkout

`server/internal/daemon/repocache/cache.go::CreateWorktree` already accepts a target branch. The daemon's `runTask` passes `task.TargetBranch` instead of the locally derived name. No new code path in the repocache — it already does the right thing (fetch, then either checkout existing remote branch or branch off `origin/HEAD`).

### Daemon: brief

When `task.IsSharedBranch` is true, the brief gets a new section appended:

> ## Shared branch
> This issue's commits go to the shared branch `<branch>`. Other issues of this feature also push there. Rules:
> - Do not `git push --force`.
> - Do not rewrite history (`git rebase -i`, `git reset --hard origin/<branch>`, `git commit --amend` after first push).
> - If the remote branch has new commits when you go to push, `git pull --rebase` first.
> - Open the PR on first push if it doesn't exist; append commits to the existing PR otherwise.

This injection lives in `server/internal/daemon/prompt.go` next to the existing brief assembly.

### Autopilot dispatch gate

`server/internal/service/autopilot.go::DispatchAutopilot` (or the equivalent function — confirmed during implementation) checks the parent feature's status before enqueueing a task. If the feature exists and is not `in_progress`, the dispatch is recorded as a skip (the existing autopilot_run audit handles this) and no row is inserted into `agent_task_queue`.

This makes the user-facing ritual "approve the PRD" map to "set `feature.status = 'in_progress'`" — already an existing PATCH endpoint.

### MCP server (`multica mcp`)

New subcommand of the existing `multica` CLI. Stdio transport only in v1.

- **Entry point**: `server/cmd/multica/cmd_mcp.go` registers `multica mcp` with cobra. Reads CLI config (token, workspace, server URL) via the same loader the other subcommands use.
- **Server core**: `server/internal/mcp/server.go` implements the stdio JSON-RPC loop, tool registration, and request dispatch. Selected SDK: `github.com/mark3labs/mcp-go` (more mature than the official Go SDK at time of writing; the choice is internal and reversible — the tool implementations don't depend on the SDK).
- **Tools** (`server/internal/mcp/tools_*.go`), one source file per logical group:
  - `tools_feature.go`: `create_feature`, `update_feature`, `approve_feature` (= PATCH status=in_progress), `set_feature_status`
  - `tools_issue.go`: `create_issue` (requires `feature_id`), `update_issue`, `set_issue_status`, `assign_issue`, `comment_on_issue`, `link_issue_dependency` (issue_id, depends_on_issue_id, type='blocks'|'related')
  - `tools_read.go`: `list_features`, `get_feature` (returns feature + child issues grouped by dependency layer + PR link), `list_issues`, `get_issue`, `list_agents` (so the model knows which agent IDs are valid assignees)
- **JSONSchema** for each tool lives next to the tool implementation, hand-written (not generated). The model sees only the fields it should fill — IDs are required where they're required, optional fields are explicitly optional.
- **HTTP layer**: every tool function calls `cli.APIClient` methods. The MCP server holds exactly one `APIClient` instance for its lifetime, constructed from the same CLI config used by every other subcommand.
- **Error surface**: REST errors are surfaced to MCP as `tool error` results with the HTTP status code and response body included in the message. The MCP server itself never returns 500 — it returns tool errors that Claude Code can show in the transcript.
- **Auth**: bearer token read from config; `MULTICA_TOKEN` env var overrides. If neither is set, the server prints a one-line setup hint to stderr and exits with code 2 before entering the stdio loop.

### Skill overrides at project level

The project ships its own `.claude/skills/` overrides for the two skills that previously wrote to `.scratch/`:

- `.claude/skills/to-prd.md`: instructs the agent to call `mcp.create_feature` with the synthesized PRD body as the description, plus `target_branch` if the conversation produced a branch name (otherwise leave NULL). Returns the created feature's identifier (e.g., `MUL-F-12`) so the user can reference it.
- `.claude/skills/to-issues.md`: instructs the agent to read the feature, decompose into issues using the same tracer-bullet vertical-slice rule as the original skill, and call `mcp.create_issue` once per slice with `feature_id` set. After all issues are created, the skill emits `mcp.link_issue_dependency` calls to encode the sequential edges (asking the user to confirm the dependency graph before persisting if it's non-obvious).

`/grill-me` is unchanged — it produces transcript context for `/to-prd` to consume.

### Web dashboard adjustments

The existing feature detail page (formerly project detail page after the rename) is restructured but not rewritten:

- The `description` field is rendered as the primary content using the existing markdown component (the one already used by `issue.description`).
- Status `planned` shows a prominent "Approve" button → PATCH `/api/features/{id}` with `status: 'in_progress'`.
- Child issues are grouped into two sections: "Ready" (no unsatisfied dependencies) and "Blocked" (with a one-line "blocked by MUL-487"). Issue rows that are currently dispatched are visually distinguished.
- Header shows `target_branch` (if set) with a link to the PR (resolved via `issue_pull_request` join — if any child issue has a linked PR, that PR is the feature's PR).

No new pages, no new components. The "approve" button is a wrapper around the existing status-update mutation.

### Out-of-scope vocabulary

- "Initiative" (Linear's umbrella above projects) is not introduced. If the user later wants to group features, that's a separate PRD.
- "Epic" is not used. The user's term is "feature".
- "Sprint" / "cycle" is not introduced. Solo workflow has no time-boxing.

## Testing Decisions

### What makes a good test in this project

Continues the established pattern from the personal-fork PRD: test external behavior, not internal structure. Backend handlers tested via real HTTP request against a test DB (`httptest`, `db_test_helper`). Frontend tested via `@testing-library/react` rendering + simulated interaction. Pure helpers tested as table-driven unit tests.

The branch resolver and the dependency gate are the two most testable pieces and should have the highest coverage relative to their size.

### Modules that get new tests

- **Branch resolver** (`server/internal/feature/branch.go`). Pure function, table-driven test. Cases:
  - feature is NULL → derived branch `issue/MUL-487-slug`, shared=false
  - feature.target_branch is NULL, issue.metadata.target_branch is NULL → derived, shared=false
  - feature.target_branch set, issue.metadata.target_branch NULL → feature's branch, shared=true
  - feature.target_branch NULL, issue.metadata.target_branch set → issue's branch, shared=false
  - both set → issue's branch wins (override), shared=false
- **SQL/Go parity test** (`server/internal/feature/branch_parity_test.go`). Asserts that for a fixture set of (feature, issue) pairs, the SQL `resolveBranch` and the Go `Resolve` produce the same result. Catches drift if either side is edited in isolation.
- **Claim handler dependency gate** (extending `server/internal/handler/daemon_test.go` or sibling). Cases:
  - Issue A depends on Issue B, B not done → A not claimed
  - Issue A depends on Issue B, B done → A claimed
  - Issue A depends on B and C, both done → A claimed
  - Issue A depends on B (done) and C (in_progress) → A not claimed
  - Issue A with `related` dependency on B (not done) → A claimed (related is non-gating)
- **Claim handler branch gate**. Cases:
  - Two queued issues sharing a feature with target_branch, slot available → first claimed, second not (until first completes)
  - Two queued issues with different branches, slot available → both claimable
  - Issue under feature.target_branch already dispatched → next issue under same feature not claimable
  - Issue with no feature_id, isolated branch → unaffected
- **Feature status gate**. Cases:
  - Issue under feature with status=planned → not claimable
  - Issue under feature with status=in_progress → claimable
  - Issue under feature with status=completed → not claimable
  - Issue with no feature → claimable (unchanged behavior)
- **Autopilot dispatch skip**. Cases:
  - Autopilot triggered for feature in status=planned → no row in agent_task_queue, skip recorded in autopilot_run
  - Autopilot triggered for feature in status=in_progress → task enqueued as today
- **MCP server tool dispatch**. Each tool function tested as a unit by injecting a fake `APIClient` that records calls. Asserts request shape and error surfacing. Does not test the stdio loop itself — that's covered by an integration test that spawns `multica mcp` as a subprocess, writes a JSON-RPC initialize + tools/list to stdin, and asserts on the response.

### Modules whose tests get adapted (not rewritten)

- Existing project handler tests (`handler/project_test.go`) renamed and updated for the new type/route names. The test logic is unchanged.
- Existing CLI tests for `multica project` renamed and updated for `multica feature`.
- Existing frontend tests for project pages updated for the new component/store names.

### Modules with no test changes

- The repocache and worktree logic — already covered by existing tests, and the rename/branch additions don't change its behavior, only what string is passed in.
- The agent CLI brief — additive change, existing tests still pass; new tests (table-driven) cover the new shared-branch section.
- Skills (`/grill-me`, `/to-prd`, `/to-issues`) — skills are markdown files consumed by Claude Code at runtime; they have no Go/TS tests. The MCP server's tool-level tests cover the API surface the skills depend on.

### Prior art

- `server/internal/handler/issue_test.go` shows the established pattern for handler tests: spin up a test DB, register the handler, fire `httptest.NewRequest`, assert on response. The new claim-handler tests follow this exact shape.
- `server/internal/handler/daemon_test.go` is the closest sibling for the claim test additions; new test cases extend the existing file.
- `server/cmd/multica/cmd_issue_test.go` shows the CLI test pattern; `cmd_feature_test.go` (renamed from `cmd_project_test.go`) follows the same setup.
- Pure-function table-driven tests like `packages/core/i18n/pick-locale.test.ts` are the model for the branch resolver test (TypeScript example but the structure transfers directly).
- No prior art for the MCP server — it's the first MCP server in the codebase. Test setup follows the standard `go test` pattern with the SDK's testing helpers if available; otherwise the stdio integration test pattern is invented here.

## Out of Scope

- **Rebranding beyond `project` → `feature`.** No other vocabulary changes in this PRD. "Workspace", "issue", "agent", "runtime", "daemon", "squad", "autopilot" stay as they are.
- **A web UI for creating features or issues.** Creation is MCP-only in v1. The web dashboard is read-only-with-approval (approve, comment, status change). This is a deliberate choice to keep the workflow concentrated in Claude Code conversations and avoid building forms.
- **An HTTP transport for the MCP server.** Stdio only in v1. Adding HTTP later is purely additive — the tool implementations don't depend on the transport.
- **MCP server hosted remotely.** The server is a local subcommand of the local `multica` binary. It cannot be reached from another machine. (If the user later wants remote agents to manipulate features, that's the HTTP transport follow-up.)
- **Renaming `issue` to something else.** Issues stay issues. "Story", "task", "ticket" are not adopted.
- **A new `prd` entity separate from `feature`.** Considered and rejected — the existing fields cover everything a PRD needs.
- **Custom dependency types beyond what `issue_dependency` already allows** (`blocks`, `blocked_by`, `related`). `related` exists but is non-gating; no new types added.
- **Dependency cycle detection at create time.** If the user creates A blocks B and B blocks A, the claim handler will simply never dispatch either, which is observable from the dashboard. A pre-flight cycle check is a nice-to-have, not v1.
- **Stacked PRs across features.** A PR maps to one feature; a feature maps to one branch (when `target_branch` is set). Cross-feature stacking (feature A's branch is based on feature B's branch) is not supported and not detected.
- **MCP server discovery from other clients.** Documentation is limited to "how to add to Claude Code". Cursor, Continue, etc. are technically supported by stdio but not explicitly documented.
- **Automatic PR creation.** The agent CLI opens the PR itself (current behavior). Multica does not create PRs on its own — it only links to them via `issue_pull_request` as the agent reports them.
- **Auto-marking issues as `done` from PR merge events.** GitHub webhook integration already exists; whether merge → done auto-transition fires is governed by existing logic and is not changed by this PRD.
- **Migration tooling for users who already have features as `.scratch/` markdown.** The user manually copies their existing `.scratch/<slug>/PRD.md` and `issues/*.md` into Multica via MCP if they want to migrate; no import script is built.
- **Multi-workspace features.** A feature belongs to one workspace, like every other entity. No cross-workspace features.

## Further Notes

### Order of execution (recommended)

The changes interact — some are blocking for others. Recommended sequence:

1. **Rename `project` → `feature`** across the codebase (schema, sqlc, handlers, CLI, frontend, MCP placeholder, reserved slugs). Pure renames have no behavior change; the test suite is the safety net. This is the largest single PR but the least risky because every change is mechanical. Lands first so subsequent PRs reference the new names.
2. **Add `feature.target_branch` column and the branch resolver module.** Schema addition + new pure module + unit tests. No behavior change yet — the column is added but nobody reads it. Lands second so the resolver exists for the claim handler to import.
3. **Add the feature-status gate to autopilot dispatch.** Single function change, integration test. Lands third so issues under unapproved features start to be skipped — verifies the "approve to start" ritual works.
4. **Add the dependency gate and branch gate to the claim handler.** The substantive scheduler change. Integration tests against a real DB. Lands fourth.
5. **Daemon-side wiring: pass `TargetBranch` and `IsSharedBranch` through the task payload, append shared-branch section to brief.** Self-contained; existing repocache logic unchanged. Lands fifth.
6. **MCP server skeleton: subcommand registered, stdio loop, no tools yet.** Lands sixth — proves the wiring works before tool surface is added.
7. **MCP tools: read tools first (`list_features`, `get_feature`, `list_issues`, `get_issue`).** Validates the API client integration without changing any state. Lands seventh.
8. **MCP tools: feature write tools (`create_feature`, `update_feature`, `approve_feature`).** Lands eighth.
9. **MCP tools: issue write tools (`create_issue`, `update_issue`, `comment_on_issue`, `assign_issue`, `set_issue_status`, `link_issue_dependency`).** Lands ninth.
10. **Skill overrides** (`.claude/skills/to-prd.md`, `.claude/skills/to-issues.md`). Lands last because they consume the MCP tools that need to exist first.
11. **Dashboard adjustments** (description-as-primary, approve button, ready-vs-blocked grouping, branch indicator). Lands in parallel with the MCP work — independent of it.

### Risk and verification

- **Rename drift.** The biggest risk is missing a `project` reference somewhere (i18n string, route segment in a server-rendered page, a test fixture). Mitigation: `make check` after the rename PR; the typecheck + Go vet + test pass is the gate.
- **SQL/Go branch resolver drift.** Mitigation: the parity test asserts equality for a fixture set. Adding a case to one side without the other breaks the test.
- **Forgotten dependency on an issue with no feature.** Mitigation: the gate uses `feature_id IS NULL OR ...` so issues without features keep the current behavior. The test suite covers this explicitly.
- **MCP server hanging on stdio.** The chosen SDK handles the stdio framing; risk is configuration error. Mitigation: integration test spawns the subprocess.
- **User approves a feature and nothing happens.** The most user-visible failure mode. Mitigation: the autopilot dispatch test asserts the skip-vs-enqueue behavior; the claim handler test asserts the dispatch when status is in_progress.

### Future follow-ups (explicitly not in this PRD)

- **HTTP transport for the MCP server**, with per-client tokens. Enables remote agents and other MCP clients (Cursor, Continue).
- **Web UI for creating features and issues.** If conversational creation via Claude Code turns out to be too friction-y for quick edits.
- **Cycle detection at dependency-link time.** Avoid the silent "neither dispatches" failure mode.
- **Auto-rebase of the shared branch when `main` advances.** Currently the agent gets a stale base if `main` has moved between issues.
- **MCP tools for `runtime` and `agent` administration.** Manage which agents are available, register a runtime, etc. — admin tools that are lower priority than the feature/issue tools.
- **Resource attachment via MCP** (the `feature_resource` rows that hold repo URLs). v1 assumes the user attaches repos via the existing CLI / dashboard; resource management in MCP is additive.
- **Bulk operations** (`create_issues_batch`, `update_issues_batch`). v1 is one-by-one; if `/to-issues` produces twenty issues, that's twenty MCP calls.
- **Templated PRDs** — reusable spec scaffolds the model can fill in.
- **A `tracer-bullet` schema** that lets `/to-issues` annotate which slice is the integration test that proves the feature works end-to-end.

## Comments
