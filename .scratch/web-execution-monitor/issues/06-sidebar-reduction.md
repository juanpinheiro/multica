# Issue 06: Reduce the sidebar to a thin rail led by the Multica mark

**Status:** `done`
**Model:** `opus`

## Parent

`.scratch/web-execution-monitor/PRD.md`

## What to build

Strip the dead operator chrome from the sidebar and reshape it into a thin icon rail led by the Multica brand mark, so the board gets the screen. Remove: the user-identity menu (single implicit user), "My Issues" (no human assignee), the "New Issue" call-to-action and its global `C` keyboard shortcut + draft-dot wiring, and the workspace switcher's "Create workspace" affordance. Reframe the switcher as "recent projects" — navigation between known workspaces only, no creation path. Reorder nav: Issues (home) · Inbox (badge = items awaiting the human) · Features · — · Agents · Autopilots · Skills · Usage · Settings.

This is regression-prone because the removed pieces are entangled (the `C` shortcut handler, the draft store, the switcher's navigation), so the agent must understand what each piece touches before removing it. Do not remove the underlying create-workspace route/flow here (the cold-start onboarding reframe is a separate, deferred PRD) — only its affordance in the switcher and the New Issue CTA in the chrome.

## Acceptance criteria

- [ ] The Multica brand mark renders at the top of the rail.
- [ ] "My Issues", the "New Issue" CTA (and its global `C` shortcut + draft dot), the "Create workspace" affordance, and the user-identity menu are gone from the sidebar.
- [ ] The switcher lists known workspaces for navigation only, with no creation path.
- [ ] Nav renders in the new order: Issues · Inbox (with badge) · Features · Agents · Autopilots · Skills · Usage · Settings.
- [ ] Removing the `C` shortcut / draft wiring leaves no dangling handlers or references (typecheck + tests pass).
- [ ] View render tests assert the brand mark and new nav are present and the removed items are absent.

## Blocked by

- None - can start immediately.

## Comments

### Key decisions

1. **Brand mark is an inlined SVG, not the `/favicon.svg` `<img>`.** `MulticaMark`
   reuses the favicon's polygon but paints it with semantic tokens
   (`fill-foreground` rect + `fill-background` glyph) so it adapts to the active
   theme. `aria-label="Multica"` + `role="img"` give it an accessible name (the
   test queries `getByRole("img", { name: /multica/i })`). The wordmark text was
   dropped — a literal "Multica" tripped `i18next/no-literal-string`, and a pure
   mark better matches "thin rail *led by the mark*". The mark leads the
   `SidebarHeader`, above the project switcher.

2. **Nav collapsed from three groups (personal/workspace/configure) to two**
   (`primaryNav` = Issues · Inbox · Features; `secondaryNav` = Agents ·
   Autopilots · Skills · Usage · Settings). The group boundary is the "—"
   separator from the PRD. The three near-identical render maps were replaced by
   a single self-contained `NavItem` (resolves its own href/active/label),
   killing the duplication.

3. **Squads and Runtimes were dropped from the rail**, matching the PRD's and
   this issue's *explicit* ordered nav list (which enumerates only the eight
   surfaces above). Their routes and `WorkspacePaths` entries are untouched —
   only the rail affordance is gone. The `runtimes` "needs update" red dot
   (`useMyRuntimesNeedUpdate`) went with it.

4. **Dead operator chrome removed:** "My Issues" nav, the "New Issue"
   `SidebarMenuButton` + its global `C` keydown effect + `DraftDot`, the
   user-identity block at the top of the switcher dropdown, and the "Create
   workspace" `DropdownMenuItem`. The switcher is now navigation-only over the
   known workspaces (the `workspaceListOptions` list). Per the issue, the
   underlying `create-workspace` modal/route was left intact — only its switcher
   affordance was removed.

5. **No dangling references.** Removing the above let me drop the imports
   `useIssueDraftStore`, `openCreateIssueWithPreference`, `useModalStore`,
   `useMyRuntimesNeedUpdate`, `ActorAvatar`, `DropdownMenuSeparator`, and the
   icons `Plus/SquarePen/CircleUser/Monitor/Users`. The now-unused i18n keys were
   pruned from `layout.json` (`nav.my_issues/squads/runtimes`,
   `sidebar.create_workspace/new_issue/new_issue_shortcut/workspace_group/configure_group`).

### Files changed

- `packages/views/layout/app-sidebar.tsx` — brand mark, `NavItem`,
  primary/secondary nav, switcher reframed to navigation-only, removed CTA/C
  shortcut/draft dot/identity block; net −109 lines.
- `packages/views/layout/app-sidebar.test.tsx` — mock `../i18n` to resolve
  selectors against `layout.json`; render `DropdownMenuContent` children; new
  `AppSidebar chrome` describe block (brand mark present, reduced nav in order,
  dead chrome absent, no create-workspace path).
- `packages/views/locales/en/layout.json` — pruned the dead nav/sidebar keys.

### Verification

- `pnpm --filter @multica/views test` → 725 passed; `@multica/web` → 10;
  `@multica/core` → 365.
- `pnpm --filter @multica/views run typecheck` → clean. Repo `pnpm typecheck`:
  the only failures are the pre-existing errors in the **untracked** throwaway
  prototype `apps/web/app/prototype/issues-monitor/page.tsx`, which Issue 08
  deletes — not touched here.
- `eslint` on the changed files → clean.

### Notes for next iteration

- Issue 07 (`AmbientProjectBar`) mounts in the dashboard shell and is
  independent of this change.
- Issue 08 deletes the prototype route and will clear the remaining prototype
  typecheck failures.
- The cold-start / no-workspace onboarding reframe is a deferred follow-up PRD
  (per the parent PRD's Out of Scope) — the `/workspaces/new` route is
  intentionally still reachable; only the switcher's create affordance was
  removed here.
