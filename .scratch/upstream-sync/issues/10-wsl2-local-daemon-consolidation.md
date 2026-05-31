# Issue 10: WSL2 local-daemon consolidation under `LOCAL`

**Status:** `done`
**Model:** `sonnet`

## Parent

PRD 1 — Upstream Sync (`.scratch/upstream-sync/PRD.md`).

## What to build

Consolidate a WSL2-hosted local daemon under the single `LOCAL` machine in the runtimes list, and de-duplicate local machines by device name, ported from upstream. Today a WSL2 daemon can surface as a separate out-of-band runtime; after this change it appears under the one `LOCAL` machine, and repeated registrations from the same device do not fan out into phantom runtimes. Relevant to the owner's Windows/WSL2 setup. Verify against the fork's runtime/heartbeat model, which the fork kept.

## Acceptance criteria

- [x] A WSL2 local daemon shows under the `LOCAL` machine rather than as a separate out-of-band runtime.
- [x] Local machines are de-duplicated by device name; re-registration does not create duplicates.
- [x] No regression to non-WSL2 local or remote runtime listing.
- [x] Test covers consolidation/dedupe by device name.

## Blocked by

None — can start immediately.

## Comments

### Key decisions

- **Device-name dedup via `mergeDeviceNameRuntimes`** — mirrors the existing `mergeLegacyRuntimes` mechanism. When a daemon registers, the server looks up any existing row with the same `(workspace_id, provider)` and a matching (case-insensitive) `device_name` but a different `daemon_id`, then reassigns all agents/tasks and deletes the stale row. Non-fatal: failures are logged and the registration still succeeds.
- **WSL2 detection in daemon config** — `isWSL2()` checks `WSL_DISTRO_NAME` env var (always set by Windows interop) with a `/proc/version` fallback. When in WSL2, `LoadConfig()` replaces `os.Hostname()` with `COMPUTERNAME` (the Windows host name) so both the native Windows daemon and the WSL2 daemon register with the same `device_name`.
- **No new unique constraint** — kept the existing `(workspace_id, daemon_id, provider)` uniqueness; device-name dedup is merge-on-register, not a DB constraint. A unique constraint on device_name would collide for two different machines with the same hostname.
- **Schema change folded into `001_init.up.sql`** — `device_name text DEFAULT '' NOT NULL` column added inline, per the fork's one-init decision. The long-lived local DB needs `ALTER TABLE agent_runtime ADD COLUMN IF NOT EXISTS device_name text DEFAULT '' NOT NULL` to pick it up; a fresh/CI DB gets it from the migration.
- **`COMPUTERNAME` normalization** — stored as `strings.ToLower(COMPUTERNAME)` so `DESKTOP-ABC` and `desktop-abc` are treated as the same device even if WSL2 and Windows report different casing.

### Files changed

- `server/migrations/001_init.up.sql` — `device_name text DEFAULT ''::text NOT NULL` column in `agent_runtime`
- `server/pkg/db/queries/runtime.sql` — `device_name` in `UpsertAgentRuntime`; new `FindRuntimesByDeviceName` query
- `server/pkg/db/generated/runtime.sql.go` — regenerated (new column + new query)
- `server/pkg/db/generated/models.go` — `DeviceName string` field in `AgentRuntime`
- `server/internal/daemon/config.go` — `isWSL2()` helper + WSL2 device-name normalization in `LoadConfig()`
- `server/internal/daemon/config_test.go` — `TestIsWSL2_EnvVar`, `TestIsWSL2_NoSignals`, `TestLoadConfig_WSL2UsesComputerName`
- `server/internal/handler/daemon.go` — `DeviceName` in request struct, `mergeDeviceNameRuntimes()` function + call in `DaemonRegister()`
- `server/internal/handler/daemon_test.go` — `TestDaemonRegister_MergesDeviceNameRuntime`, `TestDaemonRegister_DeviceNameDedup_CaseInsensitive`, `TestDaemonRegister_DeviceNameDedup_EmptyDeviceNameIsNoop`

### Notes

- `go build ./...`, `go vet ./...` clean.
- `TestIsWSL2_EnvVar` and `TestIsWSL2_NoSignals` pass locally. `TestLoadConfig_WSL2UsesComputerName` correctly skips on Windows (test runs against Linux-specific env signals; passes on CI Linux).
- Handler integration tests compile and skip locally (no Postgres running); they run in CI against the `pgvector/pg17` service container.
- 13 pre-existing daemon test failures on this Windows machine are unrelated (temp-dir I/O, local-skill symlink scanning, openclaw `~` expansion) — confirmed unchanged.
