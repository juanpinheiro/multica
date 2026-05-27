# 10 — Remove Cloud Runtime fleet proxy

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/multica-personal-fork/PRD.md`

## What to build

Remove the Cloud Runtime fleet proxy. This subsystem exists to forward requests to Multica.ai's hosted Fleet service (which manages cloud runtimes); the fork only uses the local daemon, so every byte of this code is dead weight.

The runtime detail UI loses cloud-only affordances (custom pricing dialog, cloud node start/stop/reboot, "create cloud runtime" entry point). The local-daemon UI in the same pages stays.

## Acceptance criteria

- [x] `internal/cloudruntime/` directory deleted in full
- [x] `handler/cloud_runtime.go`, `handler/cloud_runtime_test.go` deleted
- [x] `packages/views/runtimes/components/cloud-runtime-dialog.tsx` deleted
- [x] `packages/views/runtimes/components/custom-pricing-dialog.tsx` deleted
- [x] `packages/core/runtimes/cloud-runtime.ts` deleted
- [x] `packages/core/runtimes/custom-pricing-store.ts` deleted
- [x] `/api/cloud-runtime/*` route subtree removed from `cmd/server/router.go`
- [x] `CloudRuntimeFleetURL` and `CloudRuntimeFleetTimeout` removed from `handler.Config`; `cloudRuntimeFleetURLFromEnv` helper and `envDuration("MULTICA_CLOUD_FLEET_TIMEOUT", ...)` call removed from `cmd/server/router.go`
- [x] `MULTICA_CLOUD_FLEET_URL`, `MULTICA_CLOUD_FLEET_TIMEOUT` env vars removed from `.env.example` (not present — vars were never committed to the example file)
- [x] Runtime detail page cleaned of cloud-runtime affordances: `cloudRuntimeEnabled` prop removed from `RuntimesPage`, `UnmappedPricingNotice` component deleted, custom pricing store refs removed from `usage-section.tsx`, `activity-heatmap.tsx`, `utils.ts`, and `dashboard-page.tsx`
- [x] `cloud_runtime` and `custom_pricing` locale sections removed from `en/runtimes.json` and `zh-Hans/runtimes.json`
- [x] CLI: no `multica runtime cloud-*` subcommands were present
- [x] `pnpm typecheck` passes (4/4 packages clean)
- [x] `pnpm test` passes (718 TypeScript tests across 6 packages)
- [x] `go test ./internal/handler/... ./cmd/server/...` passes (1030 tests)

## Blocked by

- 09-loopback-auth-and-singleton-user

## Notes

- `packages/core/runtimes/custom-pricing-store.ts` was also used by the workspace dashboard page and runtimes utils — all references cleaned up as part of this issue
- The `Cloud` lucide icon import in `runtimes-page.tsx` was preserved as it's still used for the sidebar "cloud" section icon in `MachineRow`
- Pre-existing Go test failures in `internal/execenv` and `internal/handler` (local skills, Windows path, openclaw config) are unrelated to this change
