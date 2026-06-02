# PRD: Standalone Personal Install — `multica up` + Claude Code plugin

**Status:** `ready-for-agent`

**Parent direction:** the Initiative Runner fork (multica + Ralph + Factory Missions). See `CONTEXT.md`, `docs/adr/0001`–`0007`, and the decision this PRD implements: **`docs/adr/0008-standalone-personal-install.md`**. This PRD turns the personal/standalone deployment into a real install: one `npm i -g multica` for the runtime and one Claude Code plugin for the control plane, so a developer can run the whole Initiative Runner on their own machine with Claude Code as the planning surface.

## Problem Statement

I want to run multica on my own machine as a personal tool — Claude Code is my control plane (planning Initiatives via MCP), and a local daemon executes them to PRs. But there is no real install path. The only runtime story is dev-mode (`make start`: a Docker Postgres container + `go run ./cmd/server` + `pnpm dev:web` + a separately-launched daemon — five moving parts), and the inherited `make selfhost` Docker-Compose path points at images that do not exist in this fork. On top of that, getting Claude Code to drive a specific Workspace means manually wiring an MCP server and remembering which Workspace a folder belongs to. For a single implicit user sitting at their own machine, this is far too heavy: Docker Desktop friction (especially on Windows), a multi-process dance, and SaaS login chrome (email codes, OAuth, JWT) that does nothing for a loopback-only personal deployment.

## Solution

Ship the personal deployment as **two layers** with a clean seam (per ADR-0008): the binary owns its own install and runtime; the Claude Code plugin owns project setup, skills, and MCP.

1. **Runtime — `npm i -g multica`, then `multica up`.** A single supervisor command boots an **embedded Postgres** (no Docker), runs migrations, starts the Go server and the daemon, and spawns the existing Next.js `output: "standalone"` monitor as a Node child. `multica down` (or Ctrl-C) tears it all down in reverse order. Platform-specific native bits (Go binary + embedded Postgres) ship via per-platform npm `optionalDependencies`; the web bundle ships in the JS package.

2. **Control plane — a Claude Code plugin via the marketplace.** `claude plugin marketplace add multica-ai/multica` → `claude plugin install multica` installs the planning skills (`/to-prd`, `/to-issues`, `/triage`), the `/setup-multica` command, and one globally-declared MCP server (`multica mcp`). The MCP server learns to resolve the active **Workspace from the cwd `.multica/workspace.toml`**, so a single registration follows whatever folder Claude Code is launched in.

3. **Per-project — `/setup-multica`.** Run inside an **Umbrella** (assuming the runtime is already up), it discovers the **Repos** under the umbrella, confirms the Workspace slug and a single **Execution Mode** (worktree default / in_place), writes `.multica/workspace.toml`, updates `CLAUDE.md` context, and triggers the reconcile that creates the Workspace + Repos on the running server. The folder then "opens in the right Workspace" with skills available and the MCP bound — no manual wiring.

The two planes (ADR-0003/0004) are preserved end to end: closing the Claude Code terminal never stops an in-flight Initiative, because the daemon and the dispatched headless orchestrator run independently of the interactive session.

## User Stories

1. As a developer, I want to install the entire multica runtime with `npm i -g multica`, so that I don't have to clone the repo, install Docker, or run five dev processes by hand.
2. As a Windows developer, I want the install to work without Docker Desktop, so that I avoid its setup friction and resource cost on my machine.
3. As a developer, I want `multica up` to start Postgres, the server, the daemon, and the monitor with one command, so that I have a working stack without orchestrating processes myself.
4. As a developer, I want `multica up` to run an embedded Postgres it manages itself (its own data directory and port), so that I don't install or administer a database.
5. As a developer, I want `multica up` to run pending database migrations automatically before serving, so that a fresh install or an upgrade is ready without a manual migrate step.
6. As a developer, I want `multica up` to start each component only after its dependency is actually ready (Postgres accepting connections before the server, server healthy before the daemon), so that I never hit a race where a component dies because its dependency wasn't up yet.
7. As a developer, I want `multica up` to open (or print) the monitor URL once everything is healthy, so that I know it's ready and where to look.
8. As a developer, I want Ctrl-C (or `multica down`) to shut everything down gracefully in reverse order, so that I don't leave an orphaned Postgres, a stranded Node process, or a locked data directory behind.
9. As a developer, I want a second `multica up` to detect an already-running stack (or a stale lock) and tell me, rather than corrupting the data directory or double-binding the port, so that re-running the command is safe.
10. As a developer, I want the monitor served by the same `multica up` command (spawned Node standalone bundle), so that I get the visual execution monitor without a separate `pnpm dev:web`.
11. As a developer, I want the native binary and embedded Postgres delivered as per-platform npm `optionalDependencies`, so that I only download the artifact for my OS/arch and the install stays small and correct.
12. As a developer on an unsupported platform, I want the install to fail with a clear message naming my platform, so that I'm not left with a half-installed package that errors cryptically at `multica up`.
13. As a developer, I want to add the multica plugin to Claude Code via the marketplace (`claude plugin install multica`), so that the planning skills and the multica MCP server appear without manual config edits.
14. As a developer, I want the plugin to declare a single global `multica mcp` server, so that I register it once instead of per project.
15. As a developer, I want the `multica mcp` server to resolve the active Workspace from the nearest `.multica/workspace.toml` above my current directory, so that launching Claude Code in a project folder automatically targets that project's Workspace.
16. As a developer, when I launch Claude Code in a folder with no `.multica/workspace.toml`, I want the MCP server to fail clearly (or fall back to an explicit default), so that I understand why no Workspace is targeted instead of silently hitting the wrong one.
17. As a developer with a manifest whose slug doesn't exist on the running server, I want the MCP server to surface a precise error (slug X not found — run `/setup-multica`), so that I know the fix.
18. As a developer, I want to run `/setup-multica` inside an Umbrella and have it discover the git Repos beneath it, so that I don't enumerate them by hand.
19. As a developer, I want `/setup-multica` to cross-reference an existing `CONTEXT-MAP.md` for Repo names/descriptions, so that the discovered list matches how I already describe my projects.
20. As a developer, I want `/setup-multica` to show the discovered Repos (with their remotes) and let me confirm or trim the set before writing anything, so that I stay in control of what gets configured.
21. As a developer, I want `/setup-multica` to ask for the Workspace slug and a single Execution Mode for the Umbrella, so that the manifest reflects my intent (isolated worktree runs vs in-place runs in my real folders).
22. As a developer, I want `/setup-multica` to default Execution Mode sensibly (worktree when Repos have remotes, since worktree clones from a URL), so that the common case needs no thought.
23. As a developer, I want `/setup-multica` to write `.multica/workspace.toml` with the Workspace slug, the chosen mode, and the confirmed Repos, so that the manifest is the single source of truth the reconciler reads.
24. As a developer, I want `/setup-multica` to add/update a multica context block in `CLAUDE.md`, so that future Claude Code sessions in this folder know they're a multica Workspace and how to use it.
25. As a developer, I want `/setup-multica` to trigger the reconcile that creates the Workspace and Repos on the running server, so that the folder immediately "opens in the right Workspace" without a separate registration step.
26. As a developer, I want `/setup-multica` to assume the runtime is already up and tell me to run `multica up` if it isn't, so that the seam between binary and plugin is explicit and I get a clear next step.
27. As a developer re-running `/setup-multica` on an already-configured Umbrella, I want it to detect the existing manifest and offer to update rather than clobber it, so that I don't lose prior choices.
28. As a developer, I want the personal build to trust loopback connections as the singleton user (no login), so that I never see an email-code/OAuth/JWT login wall on my own machine.
29. As a maintainer, I want the dead SaaS login/identity chrome (email codes/Resend, Google OAuth, JWT login pages) subtracted from the personal build, so that the install doesn't ship or prompt for auth machinery it never uses.
30. As a developer, I want clear docs for the two-step install (runtime via npm, control plane via the plugin) and the per-project `/setup-multica`, so that I can go from zero to a configured Workspace by following one page.
31. As a developer, I want each load-bearing piece (MCP cwd resolution, embedded Postgres lifecycle, supervisor ordering/teardown, repo discovery) covered by tests that fail closed, so that a future change can't silently break "opens in the right Workspace" or "`multica up` just works."

## Implementation Decisions

**Embedded Postgres manager (new deep module).** A Go module that manages a self-contained Postgres: `Start(dataDir, port) → DSN`, `Stop()`, and a readiness wait. It encapsulates locating/extracting/init'ing/running a managed Postgres binary (no Docker, no system Postgres). Start is idempotent over a clean data directory and refuses to corrupt one already in use. The DSN it returns is what the server and migrations consume. Plain Postgres is sufficient — only `pgcrypto` is required by migrations, `pg_cron` is already optional (guarded `DO/EXCEPTION`), and there is no pgvector or LISTEN/NOTIFY in the runtime — so an embedded binary needs no exotic image.

**Supervisor (`multica up` / `multica down`) (new deep module + command).** Orchestrates ordered startup with readiness gating between steps: embedded Postgres → migrations → Go server (wait for `/health`) → daemon → spawn the Next.js `output: "standalone"` monitor as a Node child. On signal (or `multica down`) it tears down in reverse order, releasing the Postgres data-directory lock and reaping child processes. It detects an already-running stack / stale lock and reports rather than double-binding. The supervisor is a new `multica` subcommand; component startup reuses the existing server, migrate, and daemon entrypoints rather than reimplementing them.

**MCP cwd→Workspace resolver (new deep module — the one net-new control-plane path).** Today `multica mcp` resolves the Workspace from the CLI profile default (`resolveWorkspaceID`). This adds cwd-based resolution: on startup, walk up from the working directory with the existing `manifest.Find` to locate `.multica/workspace.toml`, parse its `workspace` slug, and resolve that slug to a Workspace on the server. Resolution order: explicit flag/env override → cwd manifest slug → (clear error if neither). The resolver is pure over its inputs (start dir, a filesystem reader, and a slug→Workspace lookup) so it can be tested without a live server. It wires into the `multica mcp` boot path so a single global MCP registration follows whatever folder Claude Code launched in.

**`/setup-multica` (new Claude Code skill/command, markdown — not Go).** A prompt-driven, per-project command shipped in the plugin, modeled on the existing `setup-matt-pocock-skills` shape (explore → present → confirm → write). It assumes `multica up` is running. Steps: discover Repos under the Umbrella (depth-1 `.git` scan + `git remote`, cross-referenced with `CONTEXT-MAP.md` when present); present the list and confirm slug + a single Execution Mode + the Repo set; write `.multica/workspace.toml`; add/update a multica context block in `CLAUDE.md`; trigger the reconcile (via the existing CLI/server reconcile path) so the Workspace + Repos are created on the running server. Execution Mode is **workspace-level** (one per Umbrella), matching the existing manifest — not per-repo. It detects an existing manifest and offers update-not-clobber. It does not write a per-project `.mcp.json` — the global MCP from the plugin resolves the Workspace from the manifest (above).

**Repo discovery.** Reuse the existing `workspace/scan` and `workspace/reconcile` packages; the only new logic is the depth-1 discovery presentation and the `CONTEXT-MAP.md` cross-reference, plus treating a Repo without a remote as **context-only** (read for planning, never an execution target) in a worktree Workspace.

**npm distribution (release engineering).** The published `multica` package is a thin JS wrapper with a `bin` shim that dispatches to the platform-specific native artifact. Per-platform packages (e.g. `multica-win32-x64`, `multica-darwin-arm64`, `multica-linux-x64`) are declared as `optionalDependencies` and carry the Go binary + embedded Postgres + migrations; the web `output: "standalone"` bundle ships in the main JS package (the esbuild / next-swc pattern). Unsupported platforms fail at install with a message naming the platform.

**Claude Code plugin scaffold.** A `.claude-plugin/marketplace.json` + `plugin.json` that declares the bundled planning skills, the `/setup-multica` command, and the global `multica mcp` server. Installable via the native marketplace mechanism the user already uses.

**Auth posture.** The personal build relies on the existing `middleware.LoopbackAuth` (loopback → singleton user, no token) and `service.BootstrapSingletonUser`. The SaaS login/identity chrome (email codes/Resend, Google OAuth, JWT login pages) is dead weight in this build and is subtracted.

## Testing Decisions

Good tests here assert **external behavior**, not internals: given inputs, does the module produce the right observable outcome (a DSN that accepts connections, a resolved Workspace ID, components that start in the right order and stop cleanly)? Avoid asserting private call sequences or struct internals.

- **MCP cwd→Workspace resolver** — table-driven tests over the pure resolution: manifest found → correct slug → Workspace ID; no manifest above cwd → clear "not found"; slug absent from the manifest → error; slug present but unknown on the server → precise "run `/setup-multica`" error; explicit flag/env override wins over the cwd manifest. Prior art: `workspace/manifest/manifest_test.go` and `workspace/reconcile/reconcile_test.go` (pure functions with injected FS/server state).
- **Embedded Postgres manager** — lifecycle tests over real behavior: `Start` brings up a Postgres that accepts a connection on the returned DSN; `Start` is idempotent / refuses an in-use data directory; `Stop` shuts down cleanly and releases the lock. Slower (boots a real PG) but catches the regressions that would break every `multica up`. Prior art: the daemon's process/lifecycle tests and existing DB-integration test patterns under `server/`.
- **Supervisor** — ordering and teardown with fake/stub process handles: the server is not started until Postgres reports ready; the daemon is not started until the server is healthy; a shutdown signal tears components down in reverse order; an already-running stack / stale lock is detected and reported rather than double-started. Prior art: existing daemon orchestration tests that use injected fakes.
- **Repo discovery (scan delta)** — tests for the new cross-reference logic only: depth-1 `.git` discovery with remotes; `CONTEXT-MAP.md` names applied to discovered Repos; a Repo without a remote classified context-only in a worktree Workspace. Prior art: `workspace/scan` and `workspace/reconcile` existing tests.

`/setup-multica` itself is a prompt-driven skill and is not unit-tested; it is validated by the reconcile it triggers (covered by the existing reconcile tests) and the manifest it writes (covered by `manifest` parse tests).

## Out of Scope

- **Per-repo Execution Mode.** Explicitly rejected for v1 — mode stays workspace-level (one per Umbrella), matching the existing manifest. Mixing worktree and in_place Repos within one Umbrella is a future change (manifest reshape + `feature.Plan` + dispatch + gate).
- **Zero-Node single binary.** The monitor is the spawned Next.js `output: "standalone"` Node bundle, not an embedded static SPA served by Go. Embedding the monitor (killing rewrites/proxy, moving locale + root redirect client-side) is a future option, not this PRD.
- **Non-npm distribution channels** (Homebrew, Scoop, GitHub Releases install scripts). npm is the v1 channel; others can follow.
- **The `multica up` supervisor installing or launching the binary for the user, or auto-installing missing agent CLIs.** The seam is deliberate: binary install + `multica up` are manual prerequisites; `/setup-multica` only configures the project.
- **Remote / multi-machine deployment, reverse proxies, non-loopback access.** This is a single-machine personal install; non-loopback access remains the existing `MULTICA_TOKEN` path, untouched here.
- **The autonomy-hardening work** (completion teardown, PR lifecycle, merge→done) — covered by `.scratch/initiative-runner-autonomy/PRD.md`, independent of install.

## Further Notes

- The only genuinely net-new code path the install requires is the **MCP cwd→Workspace resolver**; everything else reuses existing machinery (loopback auth, singleton user, `manifest`/`scan`/`reconcile`, in_place execution, cross-repo brief, the `output: "standalone"` build). This keeps the surface small and the risk concentrated in one well-tested module.
- `pg_cron` is optional in migrations, so the embedded Postgres does not need it; the token-usage rollup simply stays unscheduled on a personal install (acceptable — it is analytics).
- Decisions and glossary for this work: `docs/adr/0008-standalone-personal-install.md` and the `Repo` / `Umbrella` / `Execution Mode` terms in `CONTEXT.md`.
