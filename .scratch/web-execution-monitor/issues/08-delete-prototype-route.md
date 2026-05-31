# Issue 08: Delete the throwaway issues-monitor prototype route

**Status:** `done`
**Model:** `haiku`

## Parent

`.scratch/web-execution-monitor/PRD.md`

## What to build

Remove the throwaway prototype route now that its validated decisions (Variant A live card, phase stepper, heartbeat, counters, waiting block, thin rail) have been folded into the real board and sidebar. Delete the prototype directory `apps/web/app/prototype/issues-monitor/` and any now-empty parent `prototype/` directory. No production code references it.

## Acceptance criteria

- [ ] `apps/web/app/prototype/issues-monitor/` is deleted (and the `prototype/` parent if it becomes empty).
- [ ] No remaining references to the prototype route anywhere in the repo.
- [ ] Typecheck and build pass after removal.

## Blocked by

- Issue 01, Issue 02, Issue 03, Issue 04, Issue 05, Issue 06, Issue 07 (the validated decisions must be folded into the real surfaces first).

## Comments

### Key decisions

1. **Complete directory removal** — The `apps/web/app/prototype/` directory contained only `issues-monitor/`, which contained only `page.tsx`. Deleting the entire `prototype/` parent directory was simpler and cleaner than leaving an empty parent.

2. **No code references** — A search of the codebase (excluding node_modules and build artifacts) confirmed no references to the prototype route outside the issue tracker documentation. The route was completely isolated.

3. **Cache cleanup** — The Next.js `.next` build cache contained stale references to the deleted prototype route. Removing `.next/` and rebuilding resolved all typecheck errors.

### Files changed

- `apps/web/app/prototype/` — **deleted** (entire directory, including `issues-monitor/page.tsx`)
- `apps/web/.next/` — **deleted** (Next.js build cache, regenerated on next build)

### Verification

- ✅ Typecheck passes (all 6 packages)
- ✅ All tests pass (730 tests across @multica/views, @multica/core, @multica/web)
- ✅ No remaining code references to the prototype route
- ✅ Git status confirms the deletion
