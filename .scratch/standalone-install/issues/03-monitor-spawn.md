# Issue 03: Monitor spawn — Next.js standalone as a supervised child

**Status:** `ready-for-agent`
**Model:** `sonnet`

## Parent

`.scratch/standalone-install/PRD.md`

## What to build

Extend the supervisor (Issue 02) so `multica up` also serves the web monitor by spawning the existing Next.js `output: "standalone"` bundle as a Node child process, gated on the server being healthy, and torn down with the rest of the stack on shutdown. The monitor URL is printed once it is up.

Per ADR-0008 the monitor is the spawned standalone Node bundle (not an embedded static SPA) — the audience already has Node, and this reuses the current build with zero web rewrite. When the standalone bundle is not present (e.g. a source tree that hasn't been built), the supervisor degrades gracefully: it reports the monitor was skipped and why, and the headless stack still comes up.

## Acceptance criteria

- [ ] `multica up` spawns the Next.js standalone bundle as a child once the server is healthy, and prints the monitor URL when it is ready.
- [ ] The monitor child is torn down in reverse order with the rest of the stack on Ctrl-C / `multica down` (no stranded Node process).
- [ ] When the standalone bundle is absent, the supervisor logs a clear "monitor skipped" message and still brings up the headless stack (server + daemon + DB) successfully.
- [ ] The monitor reaches the running server (API + WS) on the same machine.
- [ ] Tests cover: monitor child started only after server health; teardown reaps the monitor child; absent-bundle path skips cleanly without failing the stack. Reuse the fake-process pattern from Issue 02.
- [ ] `make check` passes.

## Blocked by

- Issue 02 (supervisor core)
