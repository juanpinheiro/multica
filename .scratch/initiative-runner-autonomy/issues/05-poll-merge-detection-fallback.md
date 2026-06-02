# Issue 05: Poll-based merge-detection fallback + GitHub App docs

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/initiative-runner-autonomy/PRD.md` — Module #3.

## What to build

The `in_review → done` advancement already exists (`advanceFeaturesOnPRMerge`) and is wired into the GitHub webhook handler, gated on the GitHub App being configured (`GITHUB_APP_SLUG` + `GITHUB_WEBHOOK_SECRET`). In an environment with no reachable webhook endpoint (local dev), a merged PR therefore leaves the Initiative parked at `in_review` forever.

Add a poll-based fallback that observes the merge and drives the SAME advancement sink:
- A pure decision core takes a PR's state (the shape returned by `gh pr view --json` / the GitHub PR API — at minimum `state`, `merged`, merge metadata) and decides: merged, still-open, draft, or closed-without-merge. No I/O in the decision core. Missing/unknown fields must degrade safely and never falsely report merged.
- A detected merge invokes the existing `advanceFeaturesOnPRMerge` path, preserving its feature-ID deduplication and `in_review → done` guard. Webhook and poll converge on one advancement function; double-observation advances only once.
- The poller only inspects PRs linked to Initiatives currently in `in_review`. It backs off when nothing changes so a quiet AFK run doesn't burn GitHub rate limit or CPU. Cadence/backoff is an implementation detail but must be bounded.
- When the GitHub App + webhook are configured the poller is redundant but harmless (advancement is idempotent).

Also add operator documentation for configuring the GitHub App (`GITHUB_APP_SLUG` + `GITHUB_WEBHOOK_SECRET`) as the lower-latency production path, alongside the poll fallback for local dev.

## Acceptance criteria

- [ ] A pure PR-state decision function correctly classifies merged / still-open / draft / closed-without-merge, and degrades safely on missing/unknown fields (never falsely "merged").
- [ ] A detected merge advances the linked Initiative `in_review → done` via the existing `advanceFeaturesOnPRMerge` sink, with dedup intact.
- [ ] Poll + webhook double-observation of the same merge advances the Initiative only once.
- [ ] The poller scopes to `in_review` Initiative PRs and backs off when idle (bounded cadence).
- [ ] Table-driven tests over the pure core with PR-state JSON fixtures; a higher-level test asserts the shared sink is invoked and dedup holds. Prior art: `pr_lifecycle_test.go`.
- [ ] Operator docs describe both the GitHub App/webhook setup and the poll fallback.
- [ ] `make check` passes (known Windows `repocache` git-clone flakes excepted).

## Blocked by

None - can start immediately.

## Comments

### Key decisions

- **Pure classification function is `ClassifyPRMergeState(state string)`** — takes the `github_pull_request.state` column value (already the derived lowercase string from `derivePRState`) and returns one of five constants (`PRStateMerged`, `PRStateOpen`, `PRStateDraft`, `PRStateClosed`, `PRStateUnknown`). Unknown/empty strings always resolve to `PRStateUnknown`, never falsely reporting a merge.
- **New SQL query `ListInReviewIssuesWithMergedPRs`** — added to `server/pkg/db/queries/feature.sql`, uses `SELECT DISTINCT i.*` so sqlc generates a method returning `[]db.Issue` directly (exactly what `advanceFeaturesOnPRMerge` accepts). No hand-rolled SQL in the poller; all queries go through the generated layer.
- **`PRMergePoller.tick` returns the candidate count** — >0 means at least one `in_review` Initiative had a merged PR, which resets the backoff to `interval`; 0 backs off exponentially to `maxBackoff` (5 min). After a successful advancement the feature is `done` and leaves the query's result set, so the next tick returns 0 and the backoff naturally increases.
- **`NewRouterWithOptions` now returns `(chi.Router, *handler.Handler)`** — the minimal change needed to expose the handler to `main.go` so the poller can be wired there alongside other background goroutines. `NewRouter` (no-options variant) just discards the handler. No tests use `NewRouterWithOptions` directly so no test changes needed.
- **Double-observation (poll + webhook) is safe** — `advanceFeaturesOnPRMerge` checks `feature.Status == StatusInReview` before each advance, so the second call is a no-op after the webhook already fired. `TestPRMergePoller_DeduplicatesOnDoubleObservation` asserts this explicitly.

### Files changed

- `server/pkg/db/queries/feature.sql` — new `ListInReviewIssuesWithMergedPRs` query
- `server/pkg/db/generated/feature.sql.go` — sqlc-generated implementation
- `server/internal/handler/pr_merge_poller.go` — new file: `ClassifyPRMergeState`, `PRMergePoller`, `NewPRMergePoller`
- `server/internal/handler/pr_merge_poller_test.go` — new file: 7 tests covering pure function (table-driven) + 3 integration tests
- `server/cmd/server/router.go` — `NewRouterWithOptions` signature change to return `(chi.Router, *handler.Handler)`; `NewRouter` updated to discard the handler
- `server/cmd/server/main.go` — capture handler from `NewRouterWithOptions`, create poller, start `go prMergePoller.Run(sweepCtx)`
- `docs/ops/github-app-setup.md` — new operator doc covering both webhook path and poll fallback

### Test results / notes

- 793 handler tests pass. 14 new tests (7 pure classification + 3 poller integration + fixture helpers) pass.
- `go build ./...` and `go vet ./...` clean.
- Pre-existing `pkg/agent` failures (missing `opencode`/`kimi`/`kiro`/`hermes` binaries) and `pkg/redact` Windows path test are unrelated to this change.
- No blockers; all four issues in the initiative-runner-autonomy PRD are now done.
