# Issue 17: Mission Monitor — Run/Handoff timeline and DoD pass/fail

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/initiative-runner/PRD.md` — see ADR-0007.

## What to build

Per-Issue execution detail: the timeline of Runs (worker and validator) with their Handoffs (done /
left-undone / commands+exit-codes / discoveries), and the Milestone's DoD assertions shown pass/fail so
the human can see why a follow-up was created. Keep the honest-progress signals — discrete phase stepper,
heartbeat that goes "quiet" when stalled, monotonic counters — no fake percentage bar.

## Acceptance criteria

- [ ] An Issue shows its ordered Runs with role (worker/validator) and each Run's Handoff
- [ ] A Milestone shows its DoD assertions as pass/fail
- [ ] Honest-progress signals render (phase stepper, heartbeat/quiet, counters); no fake % bar
- [ ] `pnpm typecheck` and `pnpm test` pass

## Blocked by

- `08-handoff-store`
- `09-dod-and-validator`

## Comments

### Key decisions

1. **`RunHandoffPanel`** (new `packages/views/issues/components/run-handoff-panel.tsx`) — renders a completed worker Run's structured Handoff in four collapsible subsections (Done ✓, Left undone ✗, Commands with exit codes, Discoveries →). Returns null when all arrays are empty. No fake % bar — only the structured state the agent actually wrote.

2. **`IssueDodSection`** (new `packages/views/issues/components/issue-dod-section.tsx`) — a self-hiding sidebar section (mirrors `ExecutionLogSection`) that fetches `issueDodOptions(wsId, issueId)` and renders each assertion with a pass/fail/pending marker (✓/✗/○) in the matching semantic color. Returns null when no assertions exist so issues without a Milestone DoD never show the section.

3. **Role badge in `ExecutionLogSection`** — `RoleBadge` is a focused helper that renders a "Validator" chip only when `role === "validator"`; worker is the default and needs no label. Added to both `ActiveRow` and `PastRow`.

4. **Handoff expansion in `PastRow`** — fetches all issue handoffs once at the `ExecutionLogSection` level and indexes them into a `Map<runId, Handoff>` for O(1) lookup per row. Each `PastRow` with a matching handoff shows a collapsible "Handoff" toggle that expands `RunHandoffPanel` inline.

5. **Honest-progress signals already present** — the phase stepper, indeterminate shimmer, heartbeat/quiet indicator, and monotonic counters from issue-03 (`BoardCardLiveLayer`) and the `AgentLiveCard` in the issue detail were already rendering. No fake % bar was added; this issue adds only the post-run Handoff layer.

6. **TDD** — 14 unit tests written first (8 for `RunHandoffPanel`, 6 for `IssueDodSection`), all failing before implementation. All 699 views + web tests pass; typecheck clean.

### Files changed

- **New**: `packages/views/issues/components/run-handoff-panel.tsx`
- **New**: `packages/views/issues/components/run-handoff-panel.test.tsx`
- **New**: `packages/views/issues/components/issue-dod-section.tsx`
- **New**: `packages/views/issues/components/issue-dod-section.test.tsx`
- **Modified**: `packages/views/issues/components/execution-log-section.tsx` — imports `handoffListOptions`, fetches handoffs, builds `handoffByRunId` map, adds `RoleBadge`, passes `role` + `handoff` to `PastRow`, adds expandable handoff toggle
- **Modified**: `packages/views/issues/components/issue-detail.tsx` — imports and renders `IssueDodSection` in the sidebar after the execution log
- **Modified**: `packages/views/locales/en/issues.json` — added `execution_log.{role_worker, role_validator, handoff_done, handoff_left_undone, handoff_commands, handoff_discoveries, handoff_exit_ok, handoff_exit_fail, handoff_toggle}` and `detail.section_acceptance_criteria`

### Blockers / notes

None. All checks pass.
