# Issue 01: Embedded Postgres manager

**Status:** `ready-for-agent`
**Model:** `sonnet`

## Parent

`.scratch/standalone-install/PRD.md`

## What to build

A Go deep module that manages a self-contained Postgres for the personal/standalone install — no Docker, no system Postgres. It owns the full lifecycle of a managed Postgres binary (locate/extract/init/run) behind a small interface: start it against a data directory and port, wait until it is actually accepting connections, hand back a DSN, and stop it cleanly.

The DSN it returns is what the Go server and the migration runner consume. Plain Postgres is sufficient — migrations require only `pgcrypto`, `pg_cron` is already optional (guarded `DO/EXCEPTION`), and the runtime uses no pgvector and no LISTEN/NOTIFY — so no exotic image or extension is needed.

This module is consumed by the supervisor (Issue 02) but must stand alone and be verifiable in isolation.

## Acceptance criteria

- [ ] `Start` brings up a managed Postgres bound to the given data directory and port and returns a DSN that accepts a real connection.
- [ ] A readiness wait blocks until Postgres accepts connections (not just until the process spawns).
- [ ] `Start` is idempotent over a clean data directory and refuses (clear error) to run against a data directory already in use, rather than corrupting it.
- [ ] `Stop` shuts the instance down cleanly and releases the data-directory lock so a later `Start` succeeds.
- [ ] No Docker and no pre-installed system Postgres are required.
- [ ] Tests cover lifecycle behavior externally: connect-on-DSN after `Start`, idempotent/in-use `Start`, clean `Stop` + lock release. Prior art: existing DB-integration and daemon lifecycle tests under `server/`.
- [ ] `make check` passes (Go tests for this module run; on platforms where booting a real PG is impractical in CI, the test gates appropriately rather than failing spuriously).

## Blocked by

- None - can start immediately
