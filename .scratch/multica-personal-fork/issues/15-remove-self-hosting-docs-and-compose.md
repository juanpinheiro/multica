# 15 — Remove self-hosting docs and Docker compose variants

**Status:** `done`
**Model:** `haiku`

## Parent

`.scratch/multica-personal-fork/PRD.md`

## What to build

Delete the docs and Docker compose variants that exist for third-party self-hosters. The fork IS self-hosted by definition, but the existing self-hosting docs are written for an unknown reader running a SaaS-shaped product.

## Acceptance criteria

- [ ] `SELF_HOSTING.md`, `SELF_HOSTING_ADVANCED.md`, `SELF_HOSTING_AI.md` deleted
- [ ] `docker-compose.selfhost.yml` and `docker-compose.selfhost.build.yml` deleted
- [ ] `deploy/helm/` deleted in full
- [ ] `Dockerfile.web` deleted (only `Dockerfile` for the server remains)
- [ ] `docker/` directory reviewed: any `entrypoint.sh` or scripts unique to self-host variants deleted; shared entrypoint kept if still used
- [ ] README references to self-hosting docs deleted (README rewrite in 19 will catch leftovers)
- [ ] `.dockerignore` reviewed and trimmed of references to deleted files

## Blocked by

None - can start immediately

## Comments

### Files deleted

- `SELF_HOSTING.md`
- `SELF_HOSTING_ADVANCED.md`
- `SELF_HOSTING_AI.md`
- `docker-compose.selfhost.yml`
- `docker-compose.selfhost.build.yml`
- `Dockerfile.web` (only `Dockerfile` for server remains)
- `deploy/helm/` directory (entire Helm chart)

### Files modified

- `README.md` — removed `[Self-Hosting](SELF_HOSTING.md)` link from the header navigation bar

### Key decisions

- **`docker/entrypoint.sh` kept** — it's a generic shared script used by the server Dockerfile to run migrations and start the backend. Not self-host specific.
- **`.dockerignore` unchanged** — it only contains standard ignores (node_modules, build outputs, git, IDE, OS, test files, docs/) and has no references to deleted files.
- **`docker-compose.yml` preserved** — it's the minimal dev-only compose with just PostgreSQL. The full-stack self-hosting variants are deleted.

### Verification

- `pnpm typecheck` → 6/6 packages successful (cached)
- `pnpm test` → 718 tests pass across all packages
- No remaining references to deleted files in active code (only in `.scratch/` PRD/issue documentation, which is expected)

### Blockers / notes for next iteration

None — all acceptance criteria met. Issue 16 (drop i18n) can proceed independently. Issue 19 (docs rewrite) will handle the README rewrite mentioned in its AC as a final sweep.
