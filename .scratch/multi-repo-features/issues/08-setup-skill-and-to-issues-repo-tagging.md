# Issue 08: Setup skill (scan-and-suggest) + `/to-issues` repo-tagging override

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/multi-repo-features/PRD.md`

## What to build

Two project-level skill overrides under `.claude/skills/` that turn the manifest machinery and repo-aware MCP tools into a conversational workflow.

- **Setup skill** — when a session starts and no workspace resolves (Issue 06 returns the "none" state), the skill runs the repo scanner against the current directory, presents the candidate repos, and proposes a workspace named after the folder: "Found 3 repos here — backend, frontend, qa. Create workspace 'meu-produto' with these?" On confirmation it writes `.multica/workspace.toml` (pinned workspace slug + repo entries), creates the workspace, and registers the repos via the MCP tools. The scan **suggests only** — it never writes the manifest or creates anything without confirmation. A new git repo appearing in the umbrella on a later session is detected and offered for addition; a manifest repo missing on disk produces a warning.
- **`/to-issues` override** — after decomposing a feature into slices, the skill calls `list_repos`, assigns each slice to a repo from the returned menu, and **confirms the assignments and the dependency graph with the user before persisting**. It then calls `create_issue` per slice with `repo` and `feature_id` set, followed by `link_issue_dependency` for the cross-repo edges (e.g. frontend blocked by backend).

These are markdown skills consumed by Claude Code at runtime; they have no Go/TS tests. Correctness is exercised through the MCP tool tests they depend on (Issue 07).

## Acceptance criteria

- [ ] The setup skill, given no resolved workspace, runs the scanner, presents candidates, and on a single confirmation writes a valid `.multica/workspace.toml`, creates the workspace, and registers the repos.
- [ ] The skill never writes the manifest or creates a workspace without explicit confirmation.
- [ ] A new repo in the umbrella is offered for addition on the next session; a missing-on-disk repo is surfaced as a warning.
- [ ] The `/to-issues` override assigns each slice to a repo from `list_repos`, confirms with the user, then creates issues with `repo_id` and dependency edges set.
- [ ] The written manifest round-trips through the resolver (Issue 05) without error.

## Blocked by

- Issue 05 (scanner/reconciler).
- Issue 06 (resolution returning the "none" state and the reconcile-apply path).
- Issue 07 (`list_repos`, `create_issue` repo arg).

## Comments

### Iteration 1 — implemented (Sonnet)

**Key decisions**

- **Setup skill as pure guidance, not automation.** There are no `create_workspace` or `create_repo` MCP tools; workspace and repo creation happens through the manifest reconciler that runs on every session start (`applyManifest` in `workspace_resolve.go`). The setup skill writes the `.multica/workspace.toml` manifest and notes that the reconciler registers the workspace and repos automatically on the next session start. This is the correct v1 behavior and matches the PRD's intent.

- **Manifest TOML key is `[[repo]]` (not `[[repos]]`).** The `Manifest.Repos` field has `toml:"repo"`, so the array sections are `[[repo]]`. The skill uses this exact format in its template.

- **`/to-issues` is a full rewrite, not an additive patch.** The skill system has no extension mechanism; replacing the file is the only way to override the behavior. The new version adds step 2 (`list_repos` call), a **Repo** column to the slice quiz, and `repo` argument to each `create_issue` call — all other steps are preserved verbatim.

- **`to-prd.md` fixed in the same commit.** The old skill referenced `target_branch` which was removed in Issue 01. Updated to `branch_slug` with the correct description (the slug becomes `feature/<slug>` in every repo; omit to derive from the feature ID).

- **Drift detection is a separate section, not a separate flow.** Rather than a second skill, drift handling (new repo on disk / repo missing on disk) lives in a "Handling drift" section of `setup-workspace.md`. The same skill covers both fresh setup and ongoing maintenance.

**Files changed**

- `.claude/skills/setup-workspace.md` — new file: scanner flow, confirmation, manifest write, round-trip verify, drift handling.
- `.claude/skills/to-issues.md` — updated: added step 2 (`list_repos`), repo column in quiz, `repo` arg in `create_issue`, cross-repo dependency note.
- `.claude/skills/to-prd.md` — fixed: `target_branch` → `branch_slug` in step 4 and the `create_feature` call.

**Verification**

- `make check`: all checks pass (typecheck, lint, Go tests, Vitest). Skill files are markdown — no compiled artifacts.
- The written manifest format matches `manifest.Manifest` TOML struct tags (key `workspace` + `[[repo]]` arrays) and will parse without error through `manifest.Parse`.
