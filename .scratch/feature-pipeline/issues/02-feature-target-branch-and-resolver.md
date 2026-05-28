# Issue 02: Add `feature.target_branch` column + branch resolver module

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/feature-pipeline/PRD.md`

## What to build

A deep, isolated module that resolves which git branch a task should target, plus the schema column that powers it.

**Schema**: add `target_branch text NULL` to the `feature` table. NULL means "issues under this feature get isolated branches (current behavior)". Set means "all child issues converge on this branch and the claim handler will serialize them".

**Resolver module**: new package `server/internal/feature/branch.go` exposing a pure function:

```go
type FeatureForBranch struct { TargetBranch *string }
type IssueForBranch   struct { Identifier string; Metadata map[string]any }

func Resolve(i IssueForBranch, f *FeatureForBranch) (branch string, shared bool)
```

Resolution rule (priority order):
1. `f.TargetBranch` set → that branch, `shared = true`
2. else `i.Metadata["target_branch"]` set and non-empty string → that branch, `shared = false`
3. else derived `"issue/" + i.Identifier` (e.g. `issue/MUL-487`), `shared = false`

No DB, no I/O. The function is callable from anywhere and trivial to unit-test.

**SQL mirror**: the same resolution must be expressible in SQL for the claim query's `WHERE` clause in Issue 05. A SQL fragment `COALESCE(f.target_branch, i.metadata->>'target_branch', 'issue/' || i.identifier)` produces the same result. A parity test fixture-tests both implementations against the same input set and asserts equality.

## Acceptance criteria

- [ ] Migration adds `target_branch` column to `feature` table (folded into consolidated `001_init.sql` if it hasn't shipped yet, otherwise a new migration).
- [ ] `feature` create/update handlers accept `target_branch` in the request body, persist it, and return it in the response.
- [ ] `server/internal/feature/branch.go::Resolve` exists with the signature above and the resolution rule documented.
- [ ] Table-driven unit test covers: feature NULL; feature.TargetBranch NULL and issue.Metadata empty; feature.TargetBranch set; issue.Metadata.target_branch set; both set (issue wins).
- [ ] Parity test asserts SQL `COALESCE(...)` and Go `Resolve` produce identical results for a shared fixture set of (feature, issue) pairs against a test DB.
- [ ] No call sites consume `target_branch` yet — this issue is wiring + module only; behavior change comes in Issue 05.

## Blocked by

- `.scratch/feature-pipeline/issues/01-rename-project-to-feature.md`

## Comments

### Key decisions made

1. **SQL uses `NULLIF` for empty-string safety.** The spec's COALESCE fragment treats empty string as non-null (picks it), but Go's `Resolve` correctly skips empty strings. The parity test revealed this drift; both SQL (in the parity test) and Go now use `NULLIF`/`!= ""` to skip empty strings. The actual claim query in Issue 05 should use `COALESCE(NULLIF(f.target_branch,''), NULLIF(i.metadata->>'target_branch',''), 'issue/' || i.identifier)` to stay consistent.

2. **Priority: feature wins over issue metadata.** When both `feature.target_branch` and `issue.metadata["target_branch"]` are set, the feature branch wins (and `shared=true`). This matches the SQL COALESCE order from the spec.

3. **`SearchFeatures` query updated.** The dynamic search query scans into `db.Feature` explicitly; `p.target_branch` was added to both the SELECT list and the Scan call so the column is returned in search results too.

4. **DB reset required.** `make db-reset` was run to apply the consolidated migration with the new column. No separate delta migration was needed.

### Files changed

- `server/migrations/001_init.up.sql` — added `target_branch text` to the `feature` table definition
- `server/pkg/db/queries/feature.sql` — added `target_branch` to `CreateFeature` INSERT and `UpdateFeature` SET
- `server/pkg/db/generated/feature.sql.go` — regenerated via `sqlc generate` (new `TargetBranch pgtype.Text` in params/struct)
- `server/pkg/db/generated/models.go` — regenerated (new `TargetBranch pgtype.Text` on `Feature` struct)
- `server/internal/handler/feature.go` — added `TargetBranch *string` to `FeatureResponse`, `CreateFeatureRequest`, `UpdateFeatureRequest`; updated `featureToResponse`, `CreateFeature`, `UpdateFeature`, `SearchFeatures`
- `server/internal/feature/branch.go` — new file: `FeatureForBranch`, `IssueForBranch`, `Resolve`
- `server/internal/feature/branch_test.go` — new file: 10 table-driven unit tests
- `server/internal/feature/branch_parity_test.go` — new file: 6 parity tests (Go vs SQL)

### Blockers or notes for next iteration

None — all acceptance criteria satisfied:
- Migration has `target_branch text` column in `001_init.sql`
- Create/update handlers accept, persist, and return `target_branch`; `featureToResponse` includes it
- `server/internal/feature/branch.go::Resolve` exists with the specified signature and priority rules
- Unit tests cover all 5 required cases (+ empty string and wrong type edge cases)
- Parity test runs 6 cases against a real DB (skipped when DB unavailable)
- No call sites consume `target_branch` yet (wiring only)
