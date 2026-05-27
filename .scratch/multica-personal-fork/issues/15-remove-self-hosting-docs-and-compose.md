# 15 — Remove self-hosting docs and Docker compose variants

**Status:** `ready-for-agent`
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
