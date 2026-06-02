# Standalone personal install: supervisor binary, npm distribution, Claude Code plugin

The upstream self-host path assumes a multi-user SaaS shape (Docker Compose, a Postgres
image, email-code/OAuth/JWT login, a separate Next.js server). For a single implicit user
running on their own machine — with Claude Code as the control plane — that is too heavy, and
most of the auth/identity chrome is already dead weight (loopback trust + a singleton user
exist in code: `middleware.LoopbackAuth`, `service.BootstrapSingletonUser`). We ship the
personal deployment as **two layers** with a clean seam: the binary owns its own install and
runtime; the plugin owns project setup, skills, and MCP.

## 1. Runtime — one `multica` package on npm

`npm i -g multica`, then `multica up` is a **supervisor** that boots an embedded Postgres (no
Docker), the Go server, the daemon, and the existing Next.js `output: "standalone"` monitor as
a spawned Node child. Platform-specific native bits (Go binary + embedded Postgres) ship via
per-platform `optionalDependencies` (the esbuild / next-swc pattern); the web bundle ships in
the JS package.

- **npm over Docker Compose** — Docker Desktop is the biggest friction on a personal machine
  (especially Windows); the self-host compose path is also unbuilt in this fork. The audience
  already has Node/npm, and the monitor is already a Node artifact, so npm is the coherent
  single channel.
- **Spawn the Node monitor over a zero-Node embedded static SPA** — embedding would force
  rewriting the monitor (kill `rewrites`/proxy, move locale + root redirect client-side) for an
  audience that already has Node. Spawning the existing standalone bundle is zero refactor.
- **Plain Postgres suffices** — only `pgcrypto` is required; `pg_cron` is already optional
  (guarded `DO/EXCEPTION`, analytics-only); there is no pgvector and no LISTEN/NOTIFY. So an
  embedded Postgres binary works without a SQL rewrite. (SQLite stays out — sqlc is
  Postgres-dialect.)

## 2. Control plane — a Claude Code plugin via the marketplace

`claude plugin marketplace add multica-ai/multica` → `claude plugin install multica`. The plugin
bundles the planning skills (multica-aware `/to-prd`, `/to-issues`, `/triage`), the
`/setup-multica` command, and one **globally-declared MCP server** (`multica mcp`).

The MCP server is taught to resolve the active **Workspace from the cwd `.multica/workspace.toml`**
(`manifest.Find` → slug → workspace), so a single registration follows whatever folder Claude Code
is launched in. Chosen over a per-project `.mcp.json` (which works with today's profile-default
resolution but would force re-running setup per workspace).

## 3. `/setup-multica` — per-project, assumes the runtime is up

It does not install or launch the binary (manual prerequisite). On the Umbrella it: discovers
Repos under the umbrella (depth-1 `.git` scan, cross-referenced with `CONTEXT-MAP.md`), confirms
the Workspace slug + a single **Execution Mode** (worktree default / in_place), writes
`.multica/workspace.toml`, updates `CLAUDE.md` context, and triggers the reconcile that creates
the Workspace + Repos on the running server. Execution Mode is workspace-level (one per Umbrella),
matching the existing manifest — not per-repo.

## Consequences

- `multica` is a **package** (Go binary + embedded PG + web bundle + migrations), not a lone
  binary; release tooling must build per-platform artifacts.
- The MCP server gains a cwd→Workspace resolution path (new code).
- The two planes (ADR-0003/0004) are preserved: closing the Claude Code terminal never stops an
  in-flight Initiative — the daemon + dispatched headless orchestrator run independently of the
  interactive session.
- The SaaS login/identity chrome (email codes/Resend, Google OAuth, JWT) is dead weight in the
  personal build and should be subtracted.
