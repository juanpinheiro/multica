# Issue 06: Daemon — pass branch to worktree, append shared-branch warning to brief

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/feature-pipeline/PRD.md`

## What to build

Consume the `TargetBranch` and `IsSharedBranch` fields added to `Task` in Issue 05.

**Worktree checkout**: in `server/internal/daemon/daemon.go::runTask` (or the function that prepares the worktree), pass `task.TargetBranch` as the target branch to `repocache.CreateWorktree`. The repocache already does the right thing — `git fetch origin` then either `git checkout <branch>` if the branch exists on the remote, or `git checkout -b <branch>` from `origin/HEAD` if not. No changes needed inside the repocache.

**Brief assembly**: in `server/internal/daemon/prompt.go` (or wherever the brief is assembled), append a new section when `task.IsSharedBranch` is true:

```
## Shared branch

This issue's commits go to the shared branch `<branch>`. Other issues of this feature also push there. Rules:
- Do not `git push --force`.
- Do not rewrite history (`git rebase -i`, `git reset --hard origin/<branch>`, `git commit --amend` after first push).
- If the remote has new commits when you go to push, `git pull --rebase` first.
- Open the PR on first push if it doesn't exist; append commits to the existing PR otherwise.
```

When `IsSharedBranch` is false, no section is appended (no false warnings on isolated branches).

## Acceptance criteria

- [ ] Daemon passes `task.TargetBranch` to `CreateWorktree` instead of a locally derived branch name.
- [ ] When the resolved branch already exists on the remote, the daemon checks it out and the agent sees prior commits (verified by a test that pre-creates a branch with a commit and asserts the worktree contains it).
- [ ] When the resolved branch does not exist on the remote, the daemon creates it from `origin/HEAD`.
- [ ] Brief includes the "## Shared branch" section verbatim when `task.IsSharedBranch == true`.
- [ ] Brief does NOT include the section when `task.IsSharedBranch == false`.
- [ ] Table-driven test for the prompt assembly covering both cases.
- [ ] No changes inside `repocache/cache.go` — only the caller passes a different branch.

## Blocked by

- `.scratch/feature-pipeline/issues/05-branch-gate-and-task-payload.md`

## Comments

### Key decisions made

1. **Brief is the right delivery mechanism.** The worktree creation happens via the daemon's `/repo/checkout` HTTP endpoint that the agent calls — not directly in `runTask`. The agent receives `TargetBranch` and `IsSharedBranch` via the brief (CLAUDE.md/AGENTS.md written to the working directory), which instructs it to use `--ref <TargetBranch>` when checking out repos. This passes `TargetBranch` to `CreateWorktree` via `params.Ref`. The repocache's existing `resolveBaseRef` logic handles both cases (branch exists on remote → use as base; branch doesn't exist → falls through gracefully).

2. **`## Shared branch` section lives in `buildMetaSkillContent`** (the persistent CLAUDE.md/AGENTS.md content), not in `BuildPrompt` (the per-turn message). This is the right location because it's the document agents read throughout their entire task execution, not just on the first turn.

3. **Repo checkout instruction is conditionally modified.** When `IsSharedBranch` is true, the `## Repositories` section replaces the generic `Use multica repo checkout <url>` instruction with `Use multica repo checkout <url> --ref <TargetBranch>`. This only fires for shared branches because isolated derived branches (e.g. `issue/MUL-123`) don't exist on the remote yet and passing `--ref` would fail.

4. **No repocache changes needed.** The acceptance criterion "No changes inside `repocache/cache.go`" is satisfied — `resolveBaseRef` already handles the shared-branch case correctly when `params.Ref` is set to the existing branch name.

### Files changed

- `server/internal/daemon/execenv/execenv.go` — added `TargetBranch string` and `IsSharedBranch bool` to `TaskContextForEnv`
- `server/internal/daemon/execenv/runtime_config.go` — added conditional `--ref <branch>` in repos section + `## Shared branch` section before `## Issue Metadata`
- `server/internal/daemon/execenv/runtime_config_test.go` — 4 new tests: section present/absent, section content, repo instruction with shared branch, repo instruction without (isolated)
- `server/internal/daemon/daemon.go` — mapped `task.TargetBranch` and `task.IsSharedBranch` into `taskCtx` in `runTask`

### Blockers or notes for next iteration

None. All brief-related acceptance criteria satisfied:
- `## Shared branch` section present when `IsSharedBranch == true`, absent otherwise ✓
- Section content matches spec verbatim (branch name, force-push rule, rebase rule, pull-rebase rule, PR-append rule) ✓
- Repo checkout instruction mentions `--ref <branch>` only when `IsSharedBranch` is true ✓
- Table-driven tests cover both cases ✓
- No changes to `repocache/cache.go` ✓
- 783 handler tests pass, 77/78 execenv tests pass (1 pre-existing Windows failure)
