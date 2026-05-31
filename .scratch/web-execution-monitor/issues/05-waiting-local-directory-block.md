# Issue 05: waiting_local_directory block on the card (reason + holder)

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/web-execution-monitor/PRD.md`

## What to build

Make the serial in-place constraint visible on the board. A card whose agent task is parked in `waiting_local_directory` renders an **amber block** naming why it is waiting and which issue holds the umbrella lock, so the owner understands why an in-place task is serial rather than seeing it look stuck. The `deriveLiveness` `waiting` branch returns `{ reason, holderKey }` derived from the existing `wait_reason` field (and the holding task / issue key). This consumes the `waiting_local_directory` status, `wait_reason`, and `work_dir` fields that already exist on `AgentTask` — it does not add them.

## Acceptance criteria

- [ ] `deriveLiveness` returns `waiting: { reason, holderKey }` for a `waiting_local_directory` task and `null` otherwise; the task is `active: true`. Table-driven tests written first (TDD).
- [ ] The card renders an amber block with the wait reason and the holding issue key when waiting.
- [ ] The holder key resolves from the held task / issue; an unresolved holder renders the reason without a key, without crashing.
- [ ] View render test: waiting task → amber block with reason (+ holder when resolvable).

## Blocked by

- Issue 01 (establishes `deriveLiveness` and the live card surface).

## Comments

### Key decisions

1. **`deriveLiveness` fills `waiting` from task status + `ctx.holderKey`** — when `status === "waiting_local_directory"`, `waiting.reason` comes from `task.wait_reason` (falls back to `"waiting for umbrella lock"` for older backends), and `waiting.holderKey` comes from `ctx?.holderKey ?? null`. The pure function stays pure; the caller resolves the key.

2. **`LivenessCtx.holderKey` is the injection point** — added `holderKey?: string | null` to the existing context interface. Keeps derivation pure; the view layer resolves resolution.

3. **`useHolderIssueKey` extracted as a separate hook** — parses the holder task UUID from `wait_reason` via a UUID regex, looks it up in the already-fetched snapshot to get `issue_id`, then fires `issueDetailOptions` (enabled only when `issue_id` is non-null) to retrieve the identifier. Query only fires for waiting tasks — the common path is free.

4. **`WaitingBlock` replaces the `Heartbeat` for waiting tasks** — rendered conditionally inside `BoardCardLiveLayer`: `liveness.waiting ? <WaitingBlock> : <Heartbeat>`. Uses amber semantic tokens (`bg-warning/10`, `border-warning/25`, `text-warning`, `text-warning/70`). Shows "waiting · MUL-42" on the first line and the full reason below; omits the key line when `holderKey` is null.

5. **Graceful degradation** — `holderKey` is null when: the wait_reason has no embedded UUID, the holder task is not in the snapshot, or the holder issue has no identifier. The amber block renders the reason without a key and without crashing in all cases.

### Files changed

- `packages/core/tasks/derive-liveness.ts` — `holderKey` added to `LivenessCtx`; `waiting` populated from task status + ctx in `deriveLiveness`
- `packages/core/tasks/derive-liveness.test.ts` — 5 new table-driven tests for the `waiting` field (35 total)
- `packages/views/issues/components/board-card.tsx` — `UUID_RE` + `parseHolderTaskId` helpers; `useHolderIssueKey` hook; `WaitingBlock` component; `BoardCardLiveLayer` updated to render `WaitingBlock` vs `Heartbeat`; `issueDetailOptions` import added
- `packages/views/issues/components/board-card-live-layer.test.tsx` — 6 new view tests for the waiting block (23 total)

### Notes for next iteration

- Issue 06 (sidebar reduction) and Issue 07 (AmbientProjectBar) are independent and can start immediately.

### Post-merge fix (found during UI verification)

The amber waiting block never rendered end-to-end despite the unit tests passing:
`ListWorkspaceAgentTaskSnapshot` (and `ListActiveTasksByIssue`) defined "active"
as `status IN ('queued','dispatched','running')`, **excluding
`waiting_local_directory`** — so a parked in-place task never reached the board's
snapshot and the card showed nothing. The unit tests passed only because they
inject a waiting task directly into the component.

Fix: added `'waiting_local_directory'` to both active-task UI queries
(`server/pkg/db/queries/agent.sql`), regenerated sqlc, and added a regression
case to `TestListWorkspaceAgentTaskSnapshot` asserting a waiting task is
returned. Verified in the browser: a seeded waiting task now renders the amber
`waiting · MUL-xx` block with the holder key, and the "In Progress" live count
includes it. Safe for presence: `derive-presence.ts` ignores statuses outside
running/queued/dispatched, so a waiting task doesn't change the working/queued
counts.
