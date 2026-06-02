# Issue 02: Supervisor core — `multica up` / `multica down`

**Status:** `ready-for-agent`
**Model:** `opus`

## Parent

`.scratch/standalone-install/PRD.md`

## What to build

A new `multica` subcommand that boots the whole personal runtime with one command and tears it down cleanly. `multica up` orchestrates an **ordered, readiness-gated** startup: embedded Postgres (Issue 01) → run pending migrations → start the Go server (wait for `/health`) → start the daemon. `multica down` (and Ctrl-C / SIGINT) tears the components down in **reverse order**, releasing the Postgres data-directory lock and reaping child processes.

Component startup must reuse the existing server, migrate, and daemon entrypoints rather than reimplementing them. The supervisor's job is lifecycle and ordering, not business logic. The monitor (Next.js) is added in Issue 03 — this slice brings up a working headless stack (server + daemon + DB) that the CLI and MCP can talk to.

The personal build relies on the existing loopback-trust singleton auth (`middleware.LoopbackAuth` + `service.BootstrapSingletonUser`); no login is involved.

## Acceptance criteria

- [ ] `multica up` starts embedded Postgres, runs migrations, then the server, then the daemon — each only after its dependency reports ready (Postgres accepting connections before the server; server `/health` OK before the daemon).
- [ ] Once healthy, `multica up` reports the stack is ready (server URL printed).
- [ ] Ctrl-C / SIGINT and `multica down` shut every component down in reverse order, with no orphaned Postgres, stranded child process, or held data-directory lock.
- [ ] A second `multica up` against an already-running stack (or a stale lock / bound port) detects the condition and reports it clearly instead of double-binding or corrupting the data directory.
- [ ] Component startup delegates to the existing server/migrate/daemon entrypoints (no reimplementation).
- [ ] Tests cover ordering and teardown with fake/stub process handles: server not started until Postgres ready; daemon not started until server healthy; reverse-order teardown on signal; already-running / stale-lock detection. Prior art: existing daemon orchestration tests using injected fakes.
- [ ] `make check` passes.

## Blocked by

- Issue 01 (embedded Postgres manager)
