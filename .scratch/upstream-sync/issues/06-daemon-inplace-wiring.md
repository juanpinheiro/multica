# Issue 06: Daemon in-place execution wiring

**Status:** `done`
**Model:** `opus`

## Parent

PRD 2 — Workspace In-Place Execution Mode (`.scratch/upstream-sync/PRD.md`).

## What to build

Wire the in-place execution path into the daemon, tying together the mode attribute, the resolver, the validator, the locker, and the waiting status. For an `in_place` workspace, the daemon: validates the umbrella directory; acquires the umbrella lock (parking the task as `waiting_local_directory` if held); prepares each repo the task touches by checking out or creating that repo's `feature/<slug>` branch in the **real** repo; runs the agent at the umbrella root with every repo visible; and preserves the user's own `CLAUDE.md` / `AGENTS.md` / `GEMINI.md` rather than overwriting them with generated context. If a touched repo is dirty or off an expected branch, the task fails fast with a clear message rather than switching branches over in-progress work. Commit, push, and one-PR-per-repo convergence are unchanged from the multi-repo flow. When the workspace is `worktree`, none of this engages and behavior is exactly as today.

The locker keys on the umbrella real path, so in-place execution is serial per workspace; worktree workspaces keep running repos in parallel via the existing claim gate. The two serialization mechanisms must not double-block or deadlock.

## Acceptance criteria

- [x] In-place workspace runs the agent at the umbrella root with all declared repos present as children.
- [x] Each touched repo is placed on its `feature/<slug>` branch in the real repo before the run; commit/push/PR-per-repo convergence matches worktree mode.
- [x] A dirty or off-branch touched repo fails the task fast with a clear message; no branch switch occurs over in-progress work.
- [x] The umbrella path is validated (Issue 02) before running; an invalid target fails with the validator's message.
- [x] A second in-place task in the same workspace waits (`waiting_local_directory`) until the first releases; two worktree tasks in different repos still run in parallel.
- [x] The user's `CLAUDE.md` / `AGENTS.md` / `GEMINI.md` are preserved during in-place runs.
- [x] Worktree workspaces are unaffected.
- [x] Integration tests assert the serial-in-place vs parallel-worktree behavior (no deadlock) and the fail-fast preparation.

## Blocked by

- Issue 01 (Workspace execution-mode attribute)
- Issue 02 (In-place path validator)
- Issue 03 (In-place path locker)
- Issue 04 (Execution-mode resolution)
- Issue 05 (`waiting_local_directory` task status)

## Comments

### Key decisions

- **One mode gate in `runTask`, two disjoint paths.** The env-setup region of
  `daemon.runTask` now branches on `execModeIsInPlace(task.Mode)`. The worktree
  branch is the original code verbatim (Prepare/Reuse + GC-root marking +
  `InjectRuntimeConfig`), so AC7 (worktree unaffected) holds by construction —
  in-place is purely additive and never reached unless the manifest opts in. An
  empty/unknown mode normalizes to worktree via `execmode.Normalize`.

- **The fork has no daemon-side git checkout in worktree mode** — the agent runs
  `multica repo checkout` into an isolated empty workdir. In-place is the first
  path that touches a *real* repo, so the daemon-side branch preparation
  (`execenv.PrepareInPlaceRepo`) is genuinely new and scoped to in-place only.
  It fails fast (no branch switch) when the tree is dirty or HEAD is on a branch
  that is neither the default nor the target, and otherwise checks out / creates
  `feature/<slug>` in the real repo. Branch name comes from the existing
  `task.TargetBranch` (Issue 04's `feature.Resolve`), so commit/push/PR-per-repo
  convergence is identical to worktree mode.

- **Umbrella is resolved daemon-side, not sent by the server.** The umbrella is a
  local path the server can't know; only `mode` lives server-side. `resolveUmbrella`
  walks up from the issue repo's absolute local path to the nearest
  `.multica/workspace.toml`. A relative or empty repo path fails with a clear
  message (in-place requires the manifest to anchor the walk).

- **Instruction-file preservation via a non-clobbering inject.**
  `execenv.InjectRuntimeConfigPreserving` writes the runtime brief only when the
  provider's instruction file (`CLAUDE.md` / `AGENTS.md` / `GEMINI.md`) does *not*
  already exist; when the developer's own file is present it is left untouched
  and the daemon delivers the brief inline instead (`forceInlineBrief` →
  `execOpts.SystemPrompt`). Additive context (`.agent_context/`, skills,
  `.multica/feature/resources.json`) is written via `WriteInPlaceContextFiles`,
  which never touches instruction files — and writes under `.multica/feature/`,
  a distinct child of the umbrella's `.multica/` from `workspace.toml`, so it
  doesn't collide with the manifest (seeds Issue 08).

- **Locker wiring drives the serial/parallel split.** `Daemon.umbrellaLocker`
  (`inplace.NewLocker()`) is acquired in `prepareInPlace`, keyed on the umbrella
  real path, with a `WaitFunc` that posts `waiting_local_directory` via
  `client.WaitForLocalDirectory` (Issue 05). Once acquired after a wait, a second
  `StartTask` clears the parked state back to running (matches Issue 05's widened
  `StartAgentTask`). Worktree tasks never touch the locker, so they stay parallel
  via the existing per-`(repo, branch)` claim gate — the two mechanisms never
  double-block. The lock is released on `runTask` return (deferred); the agent —
  the only thing touching the real tree — is finished by then, and the completion
  report is a pure API call, so early release is safe and cannot corrupt work.

- **GC is skipped for in-place.** The in-place `Environment` has an empty
  `RootDir`, so `handleTask`'s `result.EnvRoot != ""` guard never writes
  `.gc_meta.json` into the developer's real umbrella and the GC loop never
  reclaims it.

### Files changed

- `server/internal/daemon/execenv/inplace_prep.go` (+ `_test.go`) — new
  `PrepareInPlaceRepo` (fail-fast dirty/off-branch + checkout) and git helpers.
- `server/internal/daemon/execenv/runtime_config.go` (+ `inplace_context_test.go`)
  — `InjectRuntimeConfigPreserving`, `WriteInPlaceContextFiles`, `instructionFileFor`.
- `server/internal/daemon/inplace_exec.go` (+ `_test.go`) — `resolveUmbrella`,
  `execModeIsInPlace`, and the `Daemon.prepareInPlace` orchestrator; daemon-seam
  serial-vs-parallel/no-deadlock locker test.
- `server/internal/daemon/daemon.go` — `umbrellaLocker` field + init; mode-gated
  env setup in `runTask`; `forceInlineBrief` in the inline-system-prompt gate.
- `server/internal/daemon/types.go` — `Task.Mode`.
- `server/internal/handler/agent.go` — `AgentTaskResponse.Mode`.
- `server/internal/handler/daemon.go` — claim handler projects `workspace.mode`
  onto the response via `execmode.Normalize`.

### Verification

- `go build ./...` + `go vet ./internal/...` clean.
- New unit/concurrency tests pass: `PrepareInPlaceRepo` (clean-default-creates,
  already-on-target, existing-target-checkout, dirty fail-fast, off-branch
  fail-fast, empty-target), `InjectRuntimeConfigPreserving` (preserve / write /
  prompt-only), `resolveUmbrella` (found / relative / empty / none), the
  `execModeIsInPlace` table, and the daemon-seam
  `TestUmbrellaLockerSerializesSameWorkspaceParallelizesDistinct` (serial behind
  a held umbrella, parallel on distinct umbrellas, no deadlock after release).
- `pnpm typecheck` green (no TS changes this slice; the API now returns `mode`,
  which the TS client ignores until Issue 07 adds it to the `Workspace` type).
- **DB-backed claim→daemon end-to-end not run locally** — Docker / the shared
  Postgres container is not up on this machine, so handler/integration suites
  skip via the standard `testHandler == nil` guard (same as Issues 01/05). They
  compile and `go vet` clean and run in CI. Issue 05's `inplace_waiting_test.go`
  already covers the locker→wait→release status flow against a DB in CI.
- The `internal/daemon` package has 14 pre-existing, environment-specific
  failures on this Windows box (daemon-id profile dirs, local-skill scanning of
  the real `~/.claude`, symlink-privilege tests, coalesce-window timing, GC
  mtime, openclaw `~`-config expansion). Confirmed pre-existing by reverting this
  slice and observing the identical openclaw panic on base; none touch the
  in-place files, and all new in-place tests pass.

### Notes for next iteration

- **Issue 07 (UI)** adds `mode` (`"worktree" | "in_place"`) + `waitReason` to the
  TS types and renders the read-only indicator with an enum-drift fallback. The
  API already returns `mode` (claim response) and `wait_reason` (task response).
- **Issue 08 (`.multica` namespacing)** can build on `WriteInPlaceContextFiles`,
  which already writes the in-place runtime context under `.multica/feature/`
  (distinct from `workspace.toml`); document the convention in `CLAUDE.md`.
- **Codex/OpenClaw in-place** currently fall back to the user's real
  `~/.codex` / `~/.openclaw` (no per-task `CODEX_HOME`/config is synthesized in
  the in-place path, since there is no isolated env root). Acceptable for v1 —
  in-place explicitly runs in the developer's real environment — but a follow-up
  could synthesize a side-located per-task home while keeping `WorkDir` at the
  umbrella if isolation of those caches becomes desirable.
