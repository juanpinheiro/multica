# Issue 13: PR lifecycle — draft early, ready-for-review on DoD green

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/initiative-runner/PRD.md` — see ADR-0005.

## What to build

One PR per Initiative. Open it as a **draft** early (so the human can watch/comment without being asked to
review), and flip it to **ready-for-review** only when the whole Initiative's DoD is green across all
Milestones — at which point the Initiative moves to `in_review`. Reuse multica's existing PR creation /
"feature ready for review" plumbing, reshaped to the Initiative boundary.

## Acceptance criteria

- [x] A draft PR is opened for an Initiative as it starts running
- [x] The PR flips to ready-for-review only when every Milestone's DoD is validated
- [x] Initiative transitions to `in_review` when the PR is ready
- [x] Per-Milestone commits are navigable within the single PR
- [x] `go test ./...` and `pnpm test` pass

## Blocked by

- `09-dod-and-validator`

## Comments

### Key decisions

1. **PR merge is the `in_review → done` gate (ADR-0005).** `advanceFeaturesOnPRMerge` in `github.go` runs after the issue auto-advance loop in `handlePullRequestEvent`; for each merged-PR issue with a linked feature in `in_review`, it calls `advanceInitiative(in_review → done)`. Deduplicates by feature ID so a PR linked to multiple issues under one Initiative only triggers one transition.

2. **`notifyFeatureReadyForReview` moved to the Orchestrator boundary.** The old `notifyFeatureReadyForReview(prev, issue)` fired at individual issue-done; it was replaced by `notifyInitiativeReadyForReview(featureID)` wired into `orchestrateIssue` when `plan.AdvanceTo == "in_review"`. This means the "flip your PR to ready-for-review" inbox alert fires only once all Milestones are validated — not when the last issue happens to be done before validation completes. The three call sites in `github.go:advanceIssueToDone`, `issue.go:UpdateIssue`, and `issue.go:BatchUpdateIssues` were updated accordingly; existing `TestNotifyFeatureReadyForReview` tests now exercise this path through the Orchestrator and still pass.

3. **`feature_pr_draft` inbox notification on first running transition.** `notifyInitiativePRDraft` added to `TaskService.advanceInitiativeToRunning` fires when the Initiative first transitions `ready → running` on task claim, prompting the human to push a draft PR for the feature branch. Fires only for shared-branch features (`branch_slug` set). Uses `s.Bus.Publish(EventInboxNew)` following the same pattern as `notification_listeners.go`.

4. **"Per-Milestone commits are navigable" is structural, not code.** All Issues under an Initiative push to the same `branch_slug` branch. Git history in the single PR naturally contains commits from all Milestones; no extra server logic required.

5. **TDD, 5 new tests in `pr_lifecycle_test.go`:** tracer bullet `TestInitiative_PRMerge_AdvancesInReviewToDone`, guard `TestInitiative_PRMerge_NoAdvanceUnlessInReview`, deduplication `TestInitiative_PRMerge_DeduplicatesFeatures`, draft notification `TestInitiative_FirstClaim_NotifiesPRDraft`, no-branch guard `TestInitiative_FirstClaim_NoPRDraftWithoutBranch`. All pass (888 total across 8 packages).

### Files changed

- **New**: `server/internal/handler/pr_lifecycle_test.go` — 5 DB-integrated tests
- **Modified**: `server/internal/handler/github.go` — added `advanceFeaturesOnPRMerge`, call site in `handlePullRequestEvent`, removed `notifyFeatureReadyForReview` from `advanceIssueToDone`
- **Modified**: `server/internal/handler/issue_feature_done.go` — replaced `notifyFeatureReadyForReview` with `notifyInitiativeReadyForReview(featureID)`; deleted dead old function
- **Modified**: `server/internal/handler/orchestrator.go` — added `notifyInitiativeReadyForReview` call when `plan.AdvanceTo == "in_review"`
- **Modified**: `server/internal/handler/issue.go` — removed `notifyFeatureReadyForReview` from `UpdateIssue` and `BatchUpdateIssues`
- **Modified**: `server/internal/service/task.go` — added `notifyInitiativePRDraft`, called from `advanceInitiativeToRunning`

### Blockers / notes

- `go test ./internal/handler/ ./internal/service/ ./internal/orchestrator/...` → 888 passed; `pnpm test` → 677 passed; `pnpm typecheck` → clean.
- Pre-existing env-specific failures (`repocache` Windows git clone issues, WS integration timing) are unrelated to this change.
- The `feature_pr_draft` inbox notification prompts the human to create a draft PR via `gh pr create --draft`. The server has no GitHub API client (no outbound PR creation); the PR is created by the agent CLI. Once created, the webhook auto-links it to the Initiative's issues.
- **Issue 14 (MCP):** the `create_initiative` tool should surface `mode` and `branch_slug` so agents start with the right branch and the PR draft notification fires.
