# Issue 02: In-place path validator

**Status:** `done`
**Model:** `opus`

## Parent

PRD 2 — Workspace In-Place Execution Mode (`.scratch/upstream-sync/PRD.md`).

## What to build

A pure, OS-aware validator that decides whether a directory is safe to use as an in-place execution target, ported (close to verbatim) from upstream's local-directory work. It is a leaf module with no `feature`/workspace coupling and no scheduler dependency — it takes a path and returns either ok or a typed error the daemon can forward verbatim onto a task's failure comment.

Rejection rules: non-absolute paths; system roots, the user's home directory, and Windows drive roots / UNC roots; any of those reached through a symlink (checked both literally and after resolving symlinks, so a link whose target is a banned location is also rejected); and paths that don't exist, aren't directories, or aren't readable+writable (verified with a transient probe file that is created and removed). A legitimate project directory under the user's code tree passes.

## Acceptance criteria

- [ ] Pure module: no scheduler/DB/feature dependency; deterministic given a path and the OS.
- [ ] Rejects: relative paths; `/`, `/Users`, `/home`, `/root`, and the other POSIX system roots; the user's `$HOME`; Windows drive roots (`C:\`, `G:\`, …) and UNC roots.
- [ ] Rejects a path whose symlink-resolved target is a banned location (both a user-created symlink and a directly-typed canonical alias).
- [ ] Rejects non-existent paths, non-directories, and non-writable directories; the write probe is cleaned up.
- [ ] Accepts a normal project directory under the user's code tree.
- [ ] Table-driven tests including the Windows drive-root / UNC cases; the Windows cases run on the owner's machine.

## Blocked by

None — can start immediately.
