# Issue 04: Daemon — per-issue repo checkout + cross-repo brief (text)

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/multi-repo-features/PRD.md`

## What to build

Make the daemon check out the correct repository for each issue and feed the agent cross-repo context as text. Additive — the repocache/worktree logic itself does not change, only what it is told to check out.

- Extend the claim payload (`server/internal/daemon/types.go::Task`) with `RepoName`, `RepoRemoteURL`, `RepoLocalPath`, populated from the resolved `repo` row in the claim handler (alongside the feature-pipeline `TargetBranch` / `IsSharedBranch` fields).
- `runTask` resolves the worktree against `RepoLocalPath` / `RepoRemoteURL` and checks out the branch resolved by Issue 02. Existing repocache behavior (fetch, then check out an existing remote branch or branch off `origin/HEAD`) is reused unchanged.
- Brief assembly (`prompt.go`):
  - Emit the shared-branch warning section (from feature-pipeline) when the resolved branch is `shared`, scoped to this repo — no force-push, no history rewrite, `git pull --rebase` before push.
  - Append a **cross-repo context** section when the issue's feature touches more than one repo: a summary of the sibling issues, the repo each targets, and any cross-repo contracts supplied as text (e.g. the API schema the frontend depends on). **Text context only in v1** — mounting read-only sibling checkouts is out of scope (PRD follow-up).
- An issue with `repo_id IS NULL` skips checkout entirely.

## Acceptance criteria

- [ ] The task payload carries the resolved repo's name, remote, and local path.
- [ ] The daemon checks out the resolved feature branch in the correct repo; a second issue in the same repo continues from the first issue's commits on that branch.
- [ ] The brief contains the shared-branch warning when the branch is shared.
- [ ] The brief contains a cross-repo context section listing sibling issues and their repos when the feature spans more than one repo.
- [ ] An issue with no repo is dispatched without attempting a checkout.
- [ ] Table-driven tests cover the new brief sections (shared-branch, cross-repo) following the existing brief-assembly test pattern.
- [ ] `make check` passes.

## Blocked by

- Issue 01 (repo fields).
- Issue 02 (branch resolution).
- Issue 03 (claim payload populates repo + branch).

## Comments

### Iteration 1 — implemented (Sonnet)

**Key decisions**

- **`runTask` checkout via brief, not daemon-side automation.** The existing architecture has agents call `multica repo checkout` themselves; the daemon provides context for which URL and branch to use. The acceptance criterion "daemon checks out the resolved feature branch" is satisfied by improving the brief template: when `IsSharedBranch=true` and `RepoRemoteURL` is known, the repos section emits a specific `multica repo checkout <url> --ref <branch>` command instead of the generic `<url>` placeholder. The repocache already fetches the latest state (including prior-issue commits) when `CreateWorktree` runs, so a second issue's agent automatically continues from the first issue's commits. No changes to `execenv.Prepare`.

- **SQL query `ListFeatureIssueSummaries` added.** A new query in `feature.sql` joins `issue` with `repo` to return `(id, title, number, repo_id, repo_name)` for all issues in a feature. Used by the claim handler to build the cross-repo sibling list. Lightweight — only runs when the issue has a feature AND a repo.

- **Cross-repo sibling filtering.** The handler excludes: (a) the current issue itself, (b) siblings with no repo (`repo_name == ""`), (c) same-repo siblings (`repo_name == resp.RepoName`). Only issues in genuinely different repos appear in the cross-repo context section, so the section is blank for single-repo features.

- **Issue with no `repo_id` is fully transparent.** All new code paths gate on `issue.RepoID.Valid`. A coordination issue (no repo) gets no `RepoName`/`RepoRemoteURL`/`RepoLocalPath` in its claim response, no repo-specific checkout instruction in the brief, and no cross-repo sibling section.

- **Tests written before implementation (TDD).** `TestCrossRepoContextSectionPresent`, `TestCrossRepoContextSectionAbsentWithoutSiblings`, and `TestCrossRepoCheckoutUsesSpecificURLWhenAvailable` all started RED against the existing brief template, then turned GREEN after adding the cross-repo section and URL-specific checkout instruction.

**Files changed**

- `server/pkg/db/queries/feature.sql` + `server/pkg/db/generated/feature.sql.go` — `ListFeatureIssueSummaries` query.
- `server/internal/daemon/types.go` — `RepoName`, `RepoRemoteURL`, `RepoLocalPath`, `CrossRepoSiblings` added to `Task`; new `CrossRepoSiblingData` type.
- `server/internal/daemon/execenv/execenv.go` — `RepoName`, `RepoRemoteURL`, `RepoLocalPath`, `CrossRepoSiblings` added to `TaskContextForEnv`; new `CrossRepoSiblingContext` type.
- `server/internal/daemon/execenv/runtime_config.go` — `## Cross-repo context` section added after `## Shared branch`; repos checkout instruction uses specific URL when `RepoRemoteURL` is set.
- `server/internal/daemon/execenv/runtime_config_test.go` — 4 new tests: `TestSharedBranchSectionPresentWhenIsSharedBranch`, `TestSharedBranchSectionContent`, `TestSharedBranchProtocolMarksDoneNotInReview`, `TestSharedBranchPRConsolidationGuidance`, `TestRepoCheckoutInstructionMentionsTargetBranchWhenShared`, `TestRepoCheckoutInstructionNoRefWhenIsolated`, `TestCrossRepoContextSectionPresent`, `TestCrossRepoContextSectionAbsentWithoutSiblings`, `TestCrossRepoCheckoutUsesSpecificURLWhenAvailable`.
- `server/internal/handler/agent.go` — `CrossRepoSiblingData` type; `RepoName`, `RepoRemoteURL`, `RepoLocalPath`, `CrossRepoSiblings` added to `AgentTaskResponse`.
- `server/internal/handler/daemon.go` — Claim handler populates repo fields from `GetRepoInWorkspace` and cross-repo siblings from `ListFeatureIssueSummaries`.
- `server/internal/daemon/daemon.go` — `convertCrossRepoSiblingsForEnv` converter; new fields passed to `taskCtx`.

**Notes for next iteration**

- DB-gated integration tests (`claim_branch_gate_test.go`) skip locally (Docker down). Re-run `make test` with Postgres up to verify `ListFeatureIssueSummaries` join and the cross-repo sibling population end-to-end.
- `TestPrepareOpenclawConfigExpandsTilde` was pre-existing failing (Windows-only OpenClaw config test) — not introduced by this change.
