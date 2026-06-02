# Issue 18: Mission Monitor — PR-review inbox, tripwire alerts, Mode indicator

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/initiative-runner/PRD.md` — see ADR-0005.

## What to build

Repurpose the inbox from a mentions/updates feed into the human's action queue: **PRs awaiting review**
(Initiatives in `in_review`) plus **tripwire pauses** (Initiatives in `blocked` that need attention). Show
each Initiative's **Mode** (HITL/AFK) indicator. Include the **Decision Log** view (the agent-maintained
ADR layer) once available.

## Acceptance criteria

- [x] Inbox lists Initiatives in `in_review` (PRs awaiting the human) and `blocked` (tripwire pauses)
- [x] Each Initiative surfaces its Mode (HITL/AFK)
- [x] A tripwire pause produces a visible, actionable alert
- [x] `pnpm typecheck` and `pnpm test` pass

## Blocked by

- `11-mode-and-tripwire`
- `13-pr-lifecycle`

## Comments

### Key decisions

1. **`initiative_tripwire`, `feature_ready_for_review`, `feature_pr_draft` added to `InboxItemType`** — the server already emitted these three types; the TS union just didn't know about them. Now type-safe throughout.

2. **No new fetch on item selection** — the inbox item itself carries `details.feature_id`, `details.reason`, `details.mode`, `details.branch_slug`, and `details.pr_url` so the detail panel renders from what is already in the cache. No additional query needed.

3. **Mode indicator is always visible in the Initiative detail sidebar** — `feature.mode` is `"hitl" | "afk"` on every Initiative, so the PropRow renders unconditionally (no `workspace?.mode` guard). A Tooltip explains each mode's semantics.

4. **Inbox detail panel** — initiative items (no `issue_id`) fall into the existing generic detail path. Two derived variables (`selectedInitiativeFeatureId`, `selectedPrUrl`) are computed before the JSX to avoid IIFE smell, keeping the render tree flat. "View initiative" routes to `wsPaths.featureDetail(featureId)`. "Review PR" is an external `<a>` styled inline (Button's `asChild` isn't available with Base UI).

5. **`InboxDetailLabel`** — four new switch cases: `initiative_tripwire` maps `reason` to a human label; `feature_ready_for_review` shows the branch slug; `feature_pr_draft` shows the branch slug. Fall-through to type label on missing details.

### Files changed

- `packages/core/types/inbox.ts` — added 3 new `InboxItemType` values
- `packages/views/locales/en/inbox.json` — 3 new type labels + 7 new label keys (tripwire reasons, branch, view_initiative, review_pr)
- `packages/views/locales/en/features.json` — 5 new keys under `detail` (mode_label, mode_hitl, mode_afk, *_tooltip)
- `packages/views/inbox/components/inbox-detail-label.tsx` — 3 type entries in `useTypeLabels`, 4 new switch cases
- `packages/views/inbox/components/inbox-detail-label.test.tsx` — new test file, 5 TDD tests
- `packages/views/features/components/feature-detail.tsx` — Mode PropRow after exec_mode PropRow
- `packages/views/features/components/feature-detail.test.tsx` — 2 new tests for HITL/AFK indicator
- `packages/views/inbox/components/inbox-page.tsx` — `isInitiativeItem` helper, derived variables, View initiative + Review PR CTAs in detail panel

### Blockers / notes

- 706 tests pass (82 test files), `pnpm typecheck` clean across all packages.
- The Decision Log view (mentioned in AC) is gated on issue 19, which is still `ready-for-agent`. This issue covers the inbox surface and Mode indicator; the Decision Log UI can be added alongside issue 19 when it lands.
