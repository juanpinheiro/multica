# Issue 05: `/setup-multica` command + repo discovery

**Status:** `ready-for-agent`
**Model:** `sonnet`

## Parent

`.scratch/standalone-install/PRD.md`

## What to build

A per-project, prompt-driven Claude Code command (`/setup-multica`) that configures an **Umbrella** as a multica Workspace, assuming the runtime (`multica up`) is already running. Modeled on the existing `setup-matt-pocock-skills` shape: explore → present → confirm → write. It is shipped as a skill/command (markdown), not Go, and packaged into the plugin in Issue 07.

Flow:

- **Discover Repos** under the Umbrella: a depth-1 scan for `.git` directories plus their `git remote`, cross-referenced with `CONTEXT-MAP.md` when present for names/descriptions. Reuse the existing `workspace/scan` machinery; the only new logic is the depth-1 presentation, the `CONTEXT-MAP.md` cross-reference, and treating a Repo with no remote as **context-only** (read for planning, never an execution target) in a worktree Workspace.
- **Confirm** with the user: the discovered Repo set (allow trimming), the Workspace slug, and a single **Execution Mode** for the whole Umbrella (worktree default / in_place). Default worktree when Repos have remotes (worktree clones from a URL); in_place when none.
- **Write** `.multica/workspace.toml` (workspace slug, mode, confirmed `[[repo]]` entries) and add/update a multica context block in `CLAUDE.md`.
- **Trigger the reconcile** (via the existing CLI/server reconcile path) so the Workspace + Repos are created on the running server, and the folder "opens in the right Workspace."

It does NOT write a per-project `.mcp.json` — the global MCP from the plugin resolves the Workspace from the manifest (Issue 04). If the runtime is not reachable, it stops with a clear "run `multica up` first" message. On an already-configured Umbrella it offers to update rather than clobber.

## Acceptance criteria

- [ ] Running `/setup-multica` in an Umbrella discovers the git Repos beneath it (depth-1, with remotes) and cross-references `CONTEXT-MAP.md` when present.
- [ ] It presents the discovered Repos and lets the user confirm/trim before writing anything.
- [ ] It asks for the Workspace slug and one Execution Mode, defaulting worktree when Repos have remotes and in_place when none.
- [ ] It writes `.multica/workspace.toml` (slug, mode, confirmed repos) and adds/updates a multica context block in `CLAUDE.md`.
- [ ] It triggers the reconcile so the Workspace + Repos appear on the running server.
- [ ] A Repo without a remote is recorded as context-only (not an execution target) in a worktree Workspace.
- [ ] If the runtime is unreachable, it stops with a clear "run `multica up`" message; on an existing manifest it offers update-not-clobber.
- [ ] The reconcile result and manifest are exercised by existing `reconcile` / `manifest` tests; the skill itself is validated through the reconcile it triggers (no dedicated unit suite for the prompt flow).

## Blocked by

- Issue 02 (supervisor core — needs a running server to reconcile against)
