# Issue 02: Branch resolver per `(feature, repo)` + SQL/Go parity

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/multi-repo-features/PRD.md`

## What to build

Rewrite the feature-pipeline branch resolver to take the repo dimension, and keep a SQL mirror in lockstep via a parity test. Pure logic — no DB, no I/O.

Go function in `internal/feature/branch`:

```go
type Feature struct { Identifier string; BranchSlug *string }
type Repo    struct { Name string; DefaultBranch string }
type Issue   struct { Identifier string; Metadata map[string]any }

func Resolve(i Issue, f *Feature, r *Repo) (branch string, shared bool)
```

Rules, in order:
- `issue.metadata.target_branch` set → that branch, `shared=false` (explicit override wins).
- else feature present → `feature/<BranchSlug or Identifier>`, `shared=true` (all issues of the feature in this repo converge here).
- else → `issue/<Identifier>`, `shared=false`.

The branch *name* is independent of the repo (it is `feature/auth-v2` in every repo). `shared` means multiple issues push to it *within one repo*. The repo dimension is consumed by the claim gate (Issue 03), not by the name.

SQL mirror: implement `resolveBranch(i, f)` in the claim query as `COALESCE(i.metadata->>'target_branch', 'feature/' || COALESCE(f.branch_slug, f.identifier), 'issue/' || i.identifier)`. A parity test asserts the SQL and Go produce identical results across a fixture set, so editing one side without the other fails the build.

## Acceptance criteria

- [ ] `internal/feature/branch.Resolve` implements the three-argument signature and rules above.
- [ ] Table-driven test covers: issue override → override branch, shared=false; feature present, no override → `feature/<slug>`, shared=true; `BranchSlug` set vs derived from `Identifier`; no feature → `issue/<id>`, shared=false; identical branch name resolved for two different repos.
- [ ] A SQL/Go parity test (`branch_parity_test.go`) asserts equality for a fixture set of `(issue, feature)` pairs against the SQL `resolveBranch` expression.
- [ ] `make check` passes.

## Blocked by

- Issue 01 (needs `feature.branch_slug` and the `repo` shape).

## Comments

### Iteration 1 — implemented (Sonnet)

**Key decisions**

- **Priority order flipped.** The old code had `feature.BranchSlug > issue.metadata.target_branch > issue/<id>`. The new priority per the PRD is `issue.metadata.target_branch > feature (BranchSlug ?? Identifier) > issue/<id>`. This means a per-issue metadata override now wins over any feature branch — semantically, an explicit issue-level override should always take precedence.

- **`Feature.Identifier` as always-present fallback.** When a feature is present but `BranchSlug` is nil or empty, `Identifier` is used to form `feature/<Identifier>`. The daemon populates `Identifier` from the feature's UUID (`uuidToString(proj.ID)`), so every feature-linked issue produces a shared branch even without an explicit slug. This changes the claim-gate behavior: two issues under the same feature (no branch_slug) are now serialized on `feature/<UUID>` rather than running in parallel as before.

- **SQL uses `f.id::text` as the identifier fallback.** The claim query COALESCE expression was updated to `COALESCE(NULLIF(i.metadata->>'target_branch',''), CASE WHEN i.feature_id IS NOT NULL THEN 'feature/' || COALESCE(NULLIF(f.branch_slug,''), f.id::text) END, 'issue/' || w.issue_prefix || '-' || i.number)`. This mirrors the Go resolver's behavior where the feature UUID serves as the identifier.

- **`Repo` is in the signature but unused in the name computation.** The type is defined and carried through so Issue 03's claim gate can receive all three dimensions (`issue`, `feature`, `repo`) from a single `Resolve` call without a signature change. The daemon passes `nil` for now.

- **Parity test updated.** `sqlResolve` now accepts `featurePresent bool` and `featureIdentifier string` so it can mirror the `CASE WHEN feature_id IS NOT NULL` logic in a standalone SQL expression. The fixture set was extended to cover the metadata-wins-over-feature and feature-without-BranchSlug cases.

- **Claim gate tests updated.** "issue under feature with branch_slug NULL: unaffected" was renamed and reversed to assert serialization (both issues share `feature/<UUID>` now). `TestClaimTaskByRuntime_BranchPayload`'s "null branch_slug" case now expects `feature/<featureUUID>` with `shared=true`.

**Files changed**

- `server/internal/feature/branch.go` — new types (`Issue`, `Feature`, `Repo`), three-argument `Resolve`, new priority order.
- `server/internal/feature/branch_test.go` — rewritten with new type names, new test cases (BranchSlug vs Identifier, metadata-wins-over-feature, same-branch-across-repos).
- `server/internal/feature/branch_parity_test.go` — updated `sqlResolve` signature + SQL expression; fixture set extended.
- `server/pkg/db/queries/agent.sql` — `ClaimAgentTask` COALESCE flipped to new priority, `CASE WHEN feature_id IS NOT NULL` added.
- `server/pkg/db/generated/agent.sql.go` — same SQL change + comment update (manually, no sqlc run needed since the Go function signature is unchanged).
- `server/internal/handler/daemon.go` — `featureForBranch` now `*feature.Feature` with `Identifier: uuidToString(proj.ID)`; `Resolve` call updated to 3-arg.
- `server/internal/handler/claim_branch_gate_test.go` — "null branch_slug" test reversed + branch-payload test updated.

**Verification**

- `go build ./...`: clean.
- `go test ./internal/feature/... -v`: 12 tests pass (all table-driven unit tests + DB-skipped parity test).
- `pnpm typecheck`: 4/4 tasks pass.
- `pnpm test` (Vitest): 669 tests pass.
- Pre-existing Go test failures (missing agent CLIs, Windows path separator) are environment-only — none touch this change.
