# Issue 14: Issue create/transition fixes

**Status:** `done`
**Model:** `sonnet`

## Parent

PRD 1 — Upstream Sync (`.scratch/upstream-sync/PRD.md`).

## What to build

Two related issue-flow fixes ported from upstream, grouped because they share the issue create/transition domain. Confirm each is absent locally before porting.

1. **Preserve parent through create-with-agent** — ensure an issue's parent id survives the create-with-agent code path, so sub-issue hierarchy is not lost.
2. **Child-done triggers parent's agent assignee** — when a child issue transitions to done, trigger the parent issue's agent assignee, advancing multi-step agent work.

## Acceptance criteria

- [x] A sub-issue created through the create-with-agent flow retains its parent id.
- [x] Completing a child issue triggers the parent issue's agent assignee.
- [x] Completing a child whose parent has no agent assignee is a no-op (no error).
- [x] Handler/integration tests cover both behaviors.

## Blocked by

None — can start immediately.

## Comments

### Key decisions

- **Child-done trigger was pre-existing.** `issue_child_done.go` with its full implementation (`notifyParentOfChildDone`, `dispatchParentAssigneeTrigger`, loop guards, idempotency checks) and `issue_child_done_test.go` were already present and called from `UpdateIssue`, `BatchUpdateIssues`, and the GitHub webhook path. ACs 2, 3, 4 (child-done portion) are satisfied by that existing code.

- **Create-with-agent parent: plumbed through the quick-create flow.** The `QuickCreateIssueRequest` had no `parent_issue_id` field; the agent dispatched by quick-create was never told to pass `--parent` to `multica issue create`. Added end-to-end:
  - `QuickCreateIssueRequest.ParentIssueID` — optional, validated against the workspace in the handler (same guard as `feature_id`).
  - `QuickCreateContext.ParentIssueID` — persisted in the task context JSONB; zero-omitted so old tasks round-trip cleanly.
  - `EnqueueQuickCreateTask` — new `parentIssueID pgtype.UUID` parameter; stamped into the context and logged.
  - `AgentTaskResponse.QuickCreateParentIssueID` / `Task.QuickCreateParentIssueID` — surfaced on the claim response so the daemon can forward it.
  - Daemon claim handler — reads `qc.ParentIssueID` and populates `resp.QuickCreateParentIssueID`.
  - `buildQuickCreatePrompt` — when `QuickCreateParentIssueID` is set, instructs the agent to pass `--parent "<uuid>"` (authoritative, do not infer); when absent, instructs to omit (top-level issue).

- **`--parent` CLI flag confirmed.** `cmd/multica/cmd_issue.go` already accepts `--parent` and resolves it to `parent_issue_id` in the create body, so no CLI change is needed.

- **No TaskContextForEnv change.** The parent ID feeds the per-turn prompt (`buildQuickCreatePrompt` uses `Task` directly), not the static context file (`issue_context.md`). Adding it to `TaskContextForEnv` would be premature; the prompt path is sufficient.

### Files changed

- `server/internal/handler/issue.go` — `ParentIssueID` on `QuickCreateIssueRequest`; validation + pass to enqueue in `QuickCreateIssue`.
- `server/internal/service/task.go` — `ParentIssueID` on `QuickCreateContext`; `parentIssueID pgtype.UUID` parameter on `EnqueueQuickCreateTask`.
- `server/internal/handler/agent.go` — `QuickCreateParentIssueID` on `AgentTaskResponse`.
- `server/internal/daemon/types.go` — `QuickCreateParentIssueID` on `Task`.
- `server/internal/handler/daemon.go` — populate `resp.QuickCreateParentIssueID` from quick-create context.
- `server/internal/daemon/prompt.go` — `--parent` instruction in `buildQuickCreatePrompt` when parent set; omit instruction when not.
- `server/internal/handler/daemon_test.go` — `TestClaimTask_QuickCreate_SurfacesParentIssueID` (integration, requires DB).
- `server/internal/daemon/daemon_test.go` — `TestBuildPromptQuickCreate_WithParent`, `TestBuildPromptQuickCreate_WithoutParent` (unit).
- `server/cmd/server/quick_create_subscriber_test.go` — updated two `EnqueueQuickCreateTask` call sites to pass the new parameter.

### Verification

- `go build ./...` and `go vet ./...` clean.
- `go test ./internal/daemon/ -run TestBuildPrompt` — 15 passed (includes both new quick-create parent tests).
- `go test ./internal/service/...` — 65 passed.
- `pnpm test` — 679 TS tests, 82 files passed.
- `pnpm typecheck` — 4 cached, 0 errors.
- Handler integration test (`TestClaimTask_QuickCreate_SurfacesParentIssueID`) compiles and `go vet` clean; runs against the pgvector/pg17 CI service. Docker Desktop not running locally.
- Pre-existing Windows-environment failures (git clone temp paths, OpenClaw tilde expansion, GC orphan mtime) are unrelated and confirmed unchanged.
