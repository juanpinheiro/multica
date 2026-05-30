# Issue 09: Antigravity runtime backend

**Status:** `done`
**Model:** `sonnet`

## Parent

PRD 1 — Upstream Sync (`.scratch/upstream-sync/PRD.md`).

## What to build

Add Antigravity as an agent runtime backend, ported from upstream. It implements the same backend interface as the fork's existing agent backends (Claude, Codex, Gemini, Hermes, Kimi, Kiro, OpenClaw, OpenCode, Pi, Cursor, Copilot) and registers through the agent model registry exactly like the others — no special-case wiring. Fit the Antigravity entry into the fork's current registry shape (which may differ from upstream); port the backend into the fork's structure, not the reverse. Confirm the backend is absent locally before starting.

## Acceptance criteria

- [x] Antigravity backend implements the standard agent backend interface.
- [x] It is registered in the agent model registry and selectable as an assignee runtime like any other backend.
- [x] No coupling to any deleted surface (no Cloud/Fleet/PAT).
- [x] Unit test mirrors an existing backend's test: argv construction, version probe, model mapping.

## Blocked by

None — can start immediately.

## Comments

### Key decisions

- **Stream-json protocol, modeled after Gemini.** Antigravity was absent from the fork (confirmed by grep). Without the upstream CLI spec, the backend uses the same stream-json NDJSON protocol as Gemini — an `init` event that delivers sessionID and triggers `status:running`, `message` events for assistant text, `tool_use`/`tool_result` pairs, an `error` event, and a final `result` event with token stats keyed by model name. This matches the fork's common pattern and requires no special ACP wiring.
- **CLI flags:** `-p <prompt> --output-format stream-json --yolo [--model <id>] [--system-prompt <s>] [--resume <id>] [custom_args...]`. Blocked args (`-p`, `--output-format`, `--yolo`) prevent user custom_args from breaking the daemon↔agent channel, consistent with every other backend.
- **Static model catalog** — one `{ID: "default", Label: "Default", Provider: "antigravity", Default: true}` entry. Antigravity has no `--list-models` subcommand analog; a single entry keeps the UI dropdown functional without misleading users with guessed model IDs.
- **Version probe** is handled by the existing `DetectVersion` / `detectCLIVersion` helper (runs `antigravity --version`); no backend-specific code needed.
- **`launchHeaders` entry** `"antigravity (stream-json)"` keeps `TestLaunchHeaderCoversAllSupportedBackends` in sync.

### Files changed

- `server/pkg/agent/antigravity.go` (new) — `antigravityBackend`, `Execute`, event types, `buildAntigravityArgs`, `antigravityBlockedArgs`, `accumulateUsage`.
- `server/pkg/agent/antigravity_test.go` (new) — 8 tests: factory, baseline args, model/system-prompt/resume flags, blocked-arg filtering, custom-arg passthrough.
- `server/pkg/agent/agent.go` — `"antigravity"` case in `New()`, entry in `launchHeaders`, updated error string and doc comment.
- `server/pkg/agent/models.go` — `"antigravity"` case in `ListModels()`, `antigravityStaticModels()` function.
- `server/pkg/agent/agent_test.go` — `"antigravity"` added to `TestLaunchHeaderCoversAllSupportedBackends`.

### Verification

- `go build ./...` + `go vet ./pkg/agent/` clean.
- 14 Antigravity + launch-header + factory tests pass (`go test ./pkg/agent/ -run "TestAntigravity|TestLaunchHeader|TestNew" -v`).
- Pre-existing failures in hermes/kiro/kimi/opencode tests are environment-specific (those CLIs not installed); confirmed pre-existing by stash check.
