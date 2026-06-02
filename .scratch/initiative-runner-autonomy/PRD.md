# PRD: Initiative Runner — Autonomy Hardening for Reliable AFK Runs

**Status:** `ready-for-agent`

**Parent direction:** the Initiative Runner fork (multica + Ralph + Factory Missions). See `CONTEXT.md`, `docs/adr/0001`–`0007`, and `.scratch/initiative-runner/PRD.md`. This PRD closes the autonomy gaps surfaced by the first full end-to-end run (handoff 2026-06-02) so an Initiative flipped to `ready` in AFK mode runs to a ready-for-review PR and `done` without a human babysitting the pipeline.

## Problem Statement

I flip an Initiative to `ready`, walk away, and expect the daemon to drive it to a ready-for-review PR — and, once I merge, to `done` — entirely on its own. The first real end-to-end run proved the core loop works (claim → worker → validator → DoD self-heal → milestone gate → `in_review` → PR + Decision Log), but four gaps still force me back to the keyboard:

1. **The pipeline stalls for 30 minutes per stuck task.** Twice, the worker `claude` process emitted its final summary (it had genuinely finished its work) but never exited. Because the Initiative runs serial-within (one task holds the shared branch at a time), that single hung task froze the *entire* Initiative until the 30-minute idle watchdog fired — and even then the task was misreported as `blocked`, not the successful completion it actually was. For a multi-hour AFK run this is the difference between "done overnight" and "stalled at task 2, woke up to nothing."
2. **Multi-issue Initiatives produce a messy PR history.** Single-issue Initiatives give one clean PR; multi-issue ones spawn per-task `agent/*` branches that workers self-merge into the feature branch, cluttering the PR with merge commits and intermediate branches.
3. **`in_review → done` never fires automatically in my dev environment.** The merge-detection code exists, but it only runs when a GitHub webhook is delivered — and my local setup has no public webhook endpoint, so a merged PR leaves the Initiative parked at `in_review` forever.
4. **A fat-fingered `branch_slug` silently produces a broken branch name.** Passing `feat/x` as the slug yields `feature/feat/x` with no warning, so the worker pushes to a branch I didn't intend.

## Solution

Harden the four weak points so an AFK Initiative completes without intervention:

1. **Completion-driven teardown in the claude backend.** The `result` message — not process exit — becomes the authoritative completion signal. The moment it arrives, the backend records the real disposition (completed/failed) and proactively tears the process down within a short grace window, mirroring the pattern the codex backend already uses. A hung CLI now resolves in seconds with the *correct* status, and the 30-minute idle watchdog reverts to being a true last-resort safety net.
2. **Direct-to-shared-branch pushes for multi-issue workers.** Formalize the already-decided "one PR per Initiative on `feature/<branch_slug>`" model (ADR-0005) in the execution plane: multi-issue workers commit and push straight to the shared feature branch instead of creating per-task `agent/*` branches and self-merging.
3. **A poll-based merge-detection fallback.** When GitHub webhooks aren't configured/reachable, a poller checks open Initiative PRs for merge state and drives the same `advanceFeaturesOnPRMerge` sink that the webhook uses. Plus operator docs for the GitHub App path.
4. **Branch-slug validation at the boundary.** Reject a `branch_slug` that already contains `feature/`, path separators, or characters that aren't valid in a git ref — at the MCP and HTTP boundaries, before it's ever stored or concatenated.

## User Stories

1. As an operator running an AFK Initiative, I want a worker whose `claude` process hangs after finishing to be detected as *complete* within seconds, so that a single stuck task doesn't freeze the whole serial pipeline.
2. As an operator, I want a worker that emitted its final `result` to be recorded as `completed` (or `failed`, if the result said so), so that I don't see a successful run mislabeled as `blocked`/`idle_watchdog`.
3. As an operator, I want the idle watchdog to remain a true last-resort safety net (unchanged 30-minute window), so that legitimately long, silent tool calls (`npm install`, `docker build`) are never killed prematurely.
4. As an operator, I want the same completion-robustness reasoning applied across other reader-loop-only backends (e.g. gemini, copilot), so that the same class of hang doesn't resurface on a different provider.
5. As a reviewer of a multi-issue Initiative, I want a single clean PR on the shared `feature/<branch_slug>` branch with the per-Milestone commits navigable in order, so that I can review the whole Initiative without wading through intermediate `agent/*` branches and self-merge commits.
6. As a worker executing a task under a multi-issue Initiative, I want to commit and push directly to the shared feature branch, so that I don't have to open and self-merge an `agent → feature` PR for every task.
7. As an operator without a public webhook endpoint, I want the system to detect that an Initiative's PR was merged by polling, so that the Initiative advances `in_review → done` without me PATCHing it by hand.
8. As an operator, I want the poll-based detector and the webhook to share the same `advanceFeaturesOnPRMerge` advancement logic, so that merge handling stays consistent and deduplicated regardless of how the merge was observed.
9. As an operator setting up a production environment, I want clear documentation for configuring the GitHub App (`GITHUB_APP_SLUG` + `GITHUB_WEBHOOK_SECRET`), so that I can opt into the lower-latency webhook path instead of polling.
10. As a human planning an Initiative via Claude/MCP, I want `create_initiative` to reject a `branch_slug` of `feat/x` (which would become `feature/feat/x`), so that the worker pushes to the branch I actually intended.
11. As a human updating an Initiative, I want `update_initiative` to apply the same `branch_slug` validation, so that I can't smuggle a bad slug in through an edit.
12. As a human, I want a `branch_slug` validation error to explain *why* it was rejected (contains `feature/`, contains a slash, invalid git-ref character), so that I can fix it on the first try.
13. As an operator, I want the poll fallback to back off and not hammer GitHub when nothing has changed, so that polling doesn't burn rate limit or CPU during a quiet AFK run.
14. As an operator, I want each of these fixes covered by tests that fail closed, so that a future change can't silently reintroduce the hang, the bad branch name, or a missed merge.

## Implementation Decisions

### Module #1 — Completion-driven teardown (claude backend)

- **Authoritative signal is the `result` message, not process exit.** When the backend's stdout reader receives the `result` message, it captures the final disposition there (`completed`, or `failed` when `is_error`), the output text, session id, and usage — exactly as today — and additionally records that a result has been seen.
- **Proactive teardown after a short grace window.** Once a result is seen, the backend gives the process a brief grace period (~5s) to exit cleanly on its own; if it hasn't exited, the backend cancels the run context. Cancellation closes stdout (unblocking the scanner) and lets `cmd.Wait()` return; `WaitDelay` continues to force-close any grandchild process still holding the pipe. This mirrors the codex backend, which already closes stdin and cancels its context the moment its turn-complete signal arrives ("Without this, the long-running process keeps stdout open and the reader goroutine blocks forever").
- **Disposition correctness.** A run that emitted a `result` resolves as `completed`/`failed` regardless of whether the process then hung. The teardown path must NOT route a result-bearing run through the `idle_watchdog` → `blocked` disposition. The idle watchdog stays as the safety net for runs that never emit a result at all.
- **Idle watchdog unchanged.** The 30-minute window and its in-flight-tool guard are not shortened. Shortening it would false-positive on legitimate long silent tool calls and would still mislabel a completed run.
- **Cross-backend audit.** Survey the other reader-loop-only backends (gemini, copilot, and any backend whose lifecycle is driven solely by `for scanner.Scan()` with no proactive teardown on its completion signal). Apply the same completion-driven teardown where the same hang is possible. Backends that already drive teardown from a semantic completion signal (codex) need no change.

### Module #2 — Multi-issue worker branch strategy (execution plane)

- **One shared branch per Initiative, decided by `feature.Resolve`.** `feature.Resolve` remains the single source of truth for the branch name (`feature/<branch_slug ?? identifier>` for feature-scoped issues, shared=true). No change to its contract.
- **Workers push directly to the shared branch.** For an issue that resolves to a shared feature branch, the worker's worktree tracks and pushes to that branch directly instead of creating a per-task `agent/*` branch and self-merging an `agent → feature` PR. The serial-within gate already guarantees only one task holds the shared branch at a time, so concurrent writers are not a concern within a single Initiative.
- **Per-Milestone commits stay navigable.** Because all issues push to the same branch, the single PR's git history naturally contains commits from every Milestone in order — no extra server logic, consistent with issue 13's "structural, not code" decision.
- **Single-issue path is unchanged** (it already produces a clean PR).

### Module #3 — Poll-based merge detection (webhook fallback)

- **Pure decision core.** A small, pure function takes a PR's state (the shape returned by `gh pr view --json` / the GitHub PR API — at minimum `state`, `merged`, `mergedAt`/merge-commit) and decides whether the PR is merged, still open, draft, or closed-without-merge. No I/O in the decision core.
- **Shared advancement sink.** A detected merge drives the existing `advanceFeaturesOnPRMerge` path (the same one the webhook calls), preserving its feature-ID deduplication and its `in_review → done` guard. The two observation paths (webhook, poll) converge on one advancement function.
- **Polling scope and cadence.** The poller only inspects PRs linked to Initiatives currently in `in_review`. It backs off when nothing changes so a quiet AFK run doesn't burn GitHub rate limit or CPU; the exact cadence/backoff is an implementation detail but must be bounded and configurable.
- **Opt-in / mutually safe with the webhook.** When the GitHub App + webhook are configured, the poller is redundant but harmless (advancement is idempotent via the existing dedup/guard). The poll path exists for environments with no reachable webhook endpoint (local dev). Document the webhook/GitHub App setup as the lower-latency production path.

### Module #4 — `branch_slug` validation (pure module)

- **New pure validator in the `feature` package.** `ValidateBranchSlug(slug string) error` rejects a slug that: contains the `feature/` prefix (the system adds it), contains a path separator (`/`), or contains characters not valid in a git ref name. An empty slug is valid (means "no override" — `feature.Resolve` falls back to the identifier). Clearing the slug (empty string on update) remains allowed.
- **Enforced at both boundaries.** The MCP `create_initiative` and `update_initiative` tools and the corresponding feature HTTP handler validate the slug before it is stored. On rejection they return an actionable error that names the offending condition.
- **`feature.Resolve` is not the enforcement point.** `Resolve` stays a pure name-builder; validation happens upstream so a bad slug never reaches storage or concatenation.

## Testing Decisions

A good test here asserts **external behavior**, not internal wiring: the disposition and latency of a hung-but-finished agent, the branch a worktree targets, the advancement triggered by a given PR state, and the accept/reject verdict on a slug. Tests should fail closed — if a future change reintroduces the hang, the bad branch name, or a missed merge, a test must go red.

- **#1 teardown (claude backend)** — in `claude_test.go`, drive the backend with a scripted/fake process that emits a `result` message and then *does not exit*. Assert the `Result` is delivered as `completed` (or `failed` when `is_error`) within the grace window (well under the idle-watchdog horizon), and that the disposition is NOT `idle_watchdog`/`blocked`. Add a companion case where the process exits promptly after `result` to confirm the fast path is unaffected. Prior art: existing `claude_test.go` stream-driven tests and the codex teardown tests.
- **#2 multi-issue worker branch** — test the worktree branch selection on the shared-branch (feature-scoped, multi-issue) path: the worker tracks/pushes the resolved `feature/<slug>` branch directly and does not create an `agent/*` branch. Prior art: `branch_test.go`, `branch_parity_test.go`, `target_test.go`, and the repocache worktree tests.
- **#3 poll merge detector** — table-driven tests over the pure decision core with PR-state JSON fixtures: merged, still-open, draft, closed-without-merge, missing/unknown fields (must degrade safely, never falsely report merged). A higher-level test asserts a detected merge invokes the shared advancement sink and that double-observation (poll + webhook) advances only once. Prior art: `pr_lifecycle_test.go` (the existing `advanceFeaturesOnPRMerge` tests, incl. the dedup case).
- **#4 branch_slug validator** — table-driven: accepts valid slugs (e.g. `todo-v3`, `auth`, empty), rejects `feature/x`, `feat/x`, slashes, and invalid git-ref characters; each rejection carries a reason. A boundary test confirms `create_initiative`/`update_initiative` surface the validation error. Prior art: `branch_test.go`, `target_test.go`, `tools_feature_test.go`.

All new pure modules follow the project's TDD red-green-refactor loop (the gate/dod/handoff/orchestrator precedent). Go tests create their own fixture data; verification target is `make check` (Go + TS + typecheck) with the known pre-existing Windows `repocache` git-clone flakes treated as environment noise.

## Out of Scope

- Changing the idle-watchdog window or its semantics (it stays a last-resort net).
- Reworking the validator/DoD/milestone-gate logic — the e2e proved those work.
- The `merge → done` *advancement logic itself* — it already exists (`advanceFeaturesOnPRMerge`); this PRD only adds an alternative way to *observe* the merge (poll) and documents the webhook path.
- Building a full GitHub API client in the server for outbound PR creation — PR creation stays agent-CLI-driven (`gh`), as decided in issue 13.
- Retrospective/Decision Log behavior, the `gh pr ready` draft→ready flip, and feature-branch auto-create — all already fixed and validated on v2/v3.
- Parallel-across concurrency on a single shared branch (serial-within already serializes writers within an Initiative).
- Any UI/Mission Monitor changes — these are execution-plane and control-plane backend fixes.

## Further Notes

- These four items were the explicit "remaining open" list from the first full e2e (`.scratch/initiative-runner` findings; handoff 2026-06-02). #1 is the highest-value blocker for multi-hour AFK; #2–#4 are the polish that makes an unattended run trustworthy end to end.
- The live e2e stack (server `:8080`, web `:3000`, daemon, Postgres) and the throwaway repo `juanpinheiro/multica-e2e-todo` / workspace `e2e-todo` are the established verification harness — re-run an Initiative there (with the `verify`/`agent-browser` flow) to confirm each fix in the Mission Monitor.
- Suggested build order: #4 (smallest, pure, unblocks clean branch names) → #1 (the real blocker) → #2 (branch strategy) → #3 (merge fallback). Each is independently shippable.
- When this PRD is broken into implementation issues, place them under `.scratch/initiative-runner-autonomy/issues/` numbered from `01`, continuing the Initiative Runner's tracer-bullet vertical-slice convention.
