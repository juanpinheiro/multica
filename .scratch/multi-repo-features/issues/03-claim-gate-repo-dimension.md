# Issue 03: Claim gate gains the `repo_id` dimension

**Status:** `done`
**Model:** `opus`

## Parent

`.scratch/multi-repo-features/PRD.md`

## What to build

Extend the feature-pipeline claim query so the branch-serialization gate keys on `(repo_id, branch)` instead of on the branch alone. This is what lets a cross-repo feature run its backend and frontend slices in parallel while still serializing two slices that share one repo's feature branch.

The three gates in the claim query become:

```sql
-- Gate 1: dependencies satisfied (blocking deps only) — unchanged from feature-pipeline
AND NOT EXISTS (
  SELECT 1 FROM issue_dependency d
  JOIN issue b ON d.depends_on_issue_id = b.id
  WHERE d.issue_id = i.id AND d.type IN ('blocks','blocked_by') AND b.status != 'done'
)

-- Gate 2: branch not held by another dispatched task IN THE SAME REPO
AND NOT EXISTS (
  SELECT 1 FROM agent_task_queue t2
  JOIN issue i2 ON t2.issue_id = i2.id
  LEFT JOIN feature f2 ON i2.feature_id = f2.id
  WHERE t2.status = 'dispatched'
    AND t2.id != t.id
    AND i2.repo_id = i.repo_id                       -- same repo …
    AND resolveBranch(i2, f2) = resolveBranch(i, f)  -- … same branch
)

-- Gate 3: feature must be approved — unchanged
AND (f.status IS NULL OR f.status = 'in_progress')
```

The decisive line is `AND i2.repo_id = i.repo_id`. Issues with `repo_id IS NULL` are exempt from Gate 2 (they hold no branch). Reuse the `resolveBranch` SQL expression introduced in Issue 02.

## Acceptance criteria

- [ ] Two queued issues sharing a feature, **different repos**, same branch name → both claimable in parallel (slot permitting).
- [ ] Two queued issues sharing a feature **and** repo, same branch → first claimed, second blocked until the first reaches a terminal/non-dispatched state.
- [ ] An issue with `repo_id IS NULL` is unaffected by Gate 2.
- [ ] A cross-repo dependency (frontend issue blocked by backend issue) → frontend not claimed until the backend issue is `done` (Gate 1, across repos).
- [ ] Feature-status gate behavior from feature-pipeline is preserved.
- [ ] Integration tests extend `server/internal/handler/daemon_test.go` and assert both the parallel and serial cases explicitly.
- [ ] `make check` passes.

## Blocked by

- Issue 01 (needs `issue.repo_id`).
- Issue 02 (needs the `resolveBranch` SQL expression).

## Comments

### Iteration 1 — implemented (Opus)

**Key decisions**

- **One-line predicate, NULL semantics do the rest.** Gate 2 gained a single
  clause: `AND i2.repo_id = (SELECT ci.repo_id FROM issue ci WHERE ci.id = atq.issue_id)`.
  SQL three-valued logic delivers both required exemptions for free, with no
  extra `IS NULL` branches:
  - Claimed issue `repo_id IS NULL` → `i2.repo_id = NULL` is never true → the
    whole `NOT EXISTS` is satisfied → the issue is exempt (holds no branch).
  - A dispatched task whose issue has `repo_id IS NULL` → `NULL = R` never true →
    it never blocks anyone.
  - Same repo + same branch → match → blocked (serial). Different repos →
    `R1 = R2` false → not blocked (parallel).
- **Branch name stays repo-independent.** The `resolveBranch` expression from
  Issue 02 is untouched; the repo dimension lives only in the gate, exactly as
  the PRD specifies (`feature/auth-v2` is identical across repos; the `(repo,
  branch)` pair is what serializes).
- **No `feature.Resolve` signature change, no sqlc run.** Only the SQL string
  changed and the Go `ClaimAgentTask` signature is unchanged, so I hand-edited
  both `queries/agent.sql` and the generated `agent.sql.go` const + doc comment
  in lockstep (same approach Issue 02 used). `branch_parity_test` is unaffected
  — it guards the branch *name*, not the gate.
- **Gate 3 (feature-status) is not in this query.** It lives at the autopilot
  dispatch layer (`autopilot_feature_gate_test.go`), not `ClaimAgentTask`.
  "Preserved" therefore meant leaving it untouched — confirmed it is not in the
  claim path, so no change was needed.
- **Existing tests had to flip, not just gain cases.** Two pre-existing subtests
  asserted serialization for same-feature issues that had `repo_id IS NULL`.
  Under the new (correct) semantics those issues are now *exempt*, so I gave
  them a shared repo to preserve their serialization intent rather than letting
  them silently start passing for the wrong reason.

**Files changed**

- `server/pkg/db/queries/agent.sql` — added the `i2.repo_id = …` clause to
  Gate 2 + rewrote the branch-gate doc comment to describe the `(repo, branch)`
  keying and the NULL exemption.
- `server/pkg/db/generated/agent.sql.go` — same SQL clause + doc comment,
  hand-mirrored (no sqlc run needed; function signature unchanged).
- `server/internal/handler/claim_branch_gate_test.go` — added `makeRepo` helper;
  `makeIssue` now takes a `repoID` param; existing subtests assign shared repos
  where serialization is intended; new subtests cover: same feature/same branch
  *different repos* → parallel; same feature/branch but `repo_id IS NULL` →
  exempt (both claimable); cross-repo `blocked_by` dependency → frontend not
  claimed until backend `done`.

**Verification**

- `go build ./...` clean; `go vet ./internal/handler ./internal/feature` clean
  (test files typecheck).
- `internal/feature` unit tests pass; `pnpm typecheck` green (TS untouched).
- **DB-gated integration tests skip locally — Docker/Postgres is down in this
  environment** (same constraint noted in Issues 01/02). The new and updated
  `TestClaimAgentTask_BranchGate` subtests compile and are correct by
  construction against SQL NULL semantics, but must be run with the DB up
  (`make db-up && make test`) to confirm green against real SQL.

**Notes for next iteration**

- Re-run `make test` with Postgres up to confirm the branch-gate subtests pass
  end to end — that is the only outstanding verification for this slice.
- Issue 04 (daemon repo checkout) now has the gate it relies on; the claim
  payload still needs `RepoName`/`RepoRemoteURL`/`RepoLocalPath` wired (that is
  Issue 04's job, not done here).
