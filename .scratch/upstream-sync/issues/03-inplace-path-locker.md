# Issue 03: In-place path locker

**Status:** `done`
**Model:** `opus`

## Parent

PRD 2 — Workspace In-Place Execution Mode (`.scratch/upstream-sync/PRD.md`).

## What to build

A concurrency primitive that serializes tasks sharing the same on-disk directory, ported (close to verbatim) from upstream's local-directory work. A lock is keyed on the symlink-resolved real path so two routes to the same directory collapse to one lock, and it is held for a task's whole lifetime (claim → context write → agent run → report), not just the agent execution window.

Acquiring a held lock invokes a one-shot callback with the current holder (so callers can flip the task into a waiting state) and then blocks until the lock frees or the context is cancelled. Release is idempotent. Cancellation while waiting returns without taking the lock and never wedges a future acquirer. A holder lookup exposes the current owner so callers can build a human-readable wait reason. This slice delivers the module and its tests only; wiring into the daemon is a later slice.

## Acceptance criteria

- [x] Lock keyed on the symlink-resolved real path; aliased paths collapse to one lock.
- [x] Fast-path acquire records the holder; a second acquirer on the same key blocks and fires the wait callback exactly once with the holder id.
- [x] Release frees the next waiter and is idempotent (safe to call twice / deferred).
- [x] Cancellation while waiting returns the cancellation error, does not take the lock, and does not wedge a later acquirer.
- [x] Distinct keys do not contend.
- [x] Holder lookup returns the current owner or empty.
- [x] Concurrency tests cover all the above.

## Blocked by

None — can start immediately.

## Comments

### Key decisions

- **Token-channel lock per real path.** Each directory's lock is a buffered
  channel of capacity 1 (`lock.token`): the value present means free, taking it
  means held, returning it means released. This makes `Acquire` cancellable with
  a single `select` over `token` vs `ctx.Done()` — the natural Go shape for "block
  until free or cancelled" — instead of a condition variable that can't be
  cancelled cleanly.
- **`enter` / `wait` split keeps `Acquire` a 10-line orchestrator.** `enter` (under
  the mutex) registers the participant, does the non-blocking fast-path try, and
  reports the current holder; `wait` does the blocking `select`. The wait callback
  fires exactly once, structurally, in the single slow path between them.
- **Refcounted map eviction.** `lock.refs` counts live participants (holder plus
  waiters). An entry is deleted from the map only when `refs` hits 0, so the map
  doesn't grow without bound yet a waiter never has its entry pulled out from
  under it. `refs` and `holder` are guarded by `Locker.mu`.
- **Release is fully under the mutex.** Clearing the holder, decrementing refs,
  returning the token, and evicting happen in one critical section. The token send
  never blocks (a held lock means the slot is empty), so holding the mutex across
  it is safe and eliminates a holder-cleared-but-token-not-yet-returned window
  that would otherwise let a fresh acquirer observe an empty holder spuriously.
  `sync.Once` makes the returned `ReleaseFunc` idempotent (safe to double-call or
  defer).
- **Key resolution mirrors the validator.** `resolve` uses
  `filepath.EvalSymlinks` (same primitive as `validate.go`) so symlink-aliased
  routes collapse onto one lock, falling back to the absolute/cleaned path when a
  directory can't be resolved.

### Files changed

- `server/internal/workspace/inplace/locker.go` (new) — `Locker`, `Acquire`,
  `Holder`, `WaitFunc`/`ReleaseFunc` types, and the `enter`/`wait`/`releaser`/
  `evict`/`resolve` helpers.
- `server/internal/workspace/inplace/locker_test.go` (new) — concurrency tests:
  fast-path + holder, empty-when-unlocked, second-acquirer-blocks-and-waits-once,
  idempotent release, cancellation-while-waiting (no theft, no wedge), distinct
  keys don't contend, symlink aliases collapse.

### Verification

- `go vet` clean; `go build ./...` OK.
- `go test ./internal/workspace/inplace/` passes, including a 200× repeat over the
  concurrency tests with no flakes. The `-race` detector is unavailable on the
  owner's machine (no cgo C compiler); CI runs with cgo and will exercise it.
- The symlink-alias collapse test skips on the owner's Windows machine (symlink
  creation needs privilege not held) — the same skip pattern as the validator's
  existing symlink test in `validate_test.go`; it runs on POSIX/CI.

### Notes for next iteration

- This slice is the module only. Issue 05 (`waiting_local_directory` status) wires
  the `WaitFunc` callback to post the waiting status with a reason naming the
  holder (`Holder(dir)` supplies the owner id). Issue 06 (daemon in-place wiring)
  holds the lock for the task's whole lifetime — `Acquire` at claim, the returned
  `ReleaseFunc` deferred through report — keyed on the umbrella real path so
  in-place execution is serial per workspace while worktree repos stay parallel
  via the existing claim gate.
