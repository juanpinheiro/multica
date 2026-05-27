# 03 — Delete apps/mobile (Expo / React Native iOS app)

**Status:** `done`

## Parent

`.scratch/multica-personal-fork/PRD.md`

## What to build

Delete `apps/mobile/` and all mobile-specific scripts, dependencies, and documentation. Mobile is locked to old React Native versions (Expo SDK lag); removing it frees the pnpm catalog to use a single React version aligned with what Next.js needs.

## Acceptance criteria

- [x] `apps/mobile/` directory deleted in full (including `apps/mobile/CLAUDE.md`)
- [x] All `dev:mobile*` and `ios:mobile*` scripts removed from root `package.json`
- [x] `expo`, `react-native`, and the React version pin (`react@19.2.0`) direct dependencies removed from root `package.json`
- [x] Mobile-related catalog entries removed from `pnpm-workspace.yaml` *(none existed — catalog never carried Expo / RN entries; the mobile-locked versions lived in root `dependencies`, which is removed)*
- [x] "Mobile-specific Rules" section deleted from `CLAUDE.md`
- [x] `apps/mobile` line deleted from the architecture list in `CLAUDE.md`
- [x] References to `apps/mobile/CLAUDE.md` in root `CLAUDE.md` removed
- [x] `pnpm install` succeeds; catalog no longer carries mobile-locked versions
- [x] `pnpm typecheck` and `pnpm test` pass

## Blocked by

None - can start immediately

## Comments

### Key decisions

- **Mobile-locked deps lived in root `dependencies`, not the catalog.** The PRD and AC assumed catalog entries for Expo / RN, but inspection showed the catalog never carried any of them (catalog versions like `react: 19.2.3` are aligned with Next.js). The actually-pinned `react@19.2.0`, `react-native@0.83.6`, `expo@~55.0.23` were root-level direct deps. Removed the entire `dependencies` block from root `package.json` — there's no other consumer at the root level now.
- **`@xmldom/xmldom` pnpm override removed.** It was a defensive `^0.8.13` override for a transitive dep; nothing in the kept code imports it directly (grep confirms it's only in `pnpm-lock.yaml` as a transitive). Letting the original constraint resolve naturally is simpler than carrying a pin nobody needs.
- **Turbo `dev:staging` task definition removed.** It existed in `turbo.json` only to give mobile a non-cached, persistent task. No other package defines a `dev:staging` script; the task definition is dead.
- **Turbo `--filter='!@multica/mobile'` removed from root scripts.** `build`, `typecheck`, `test`, `lint` no longer carry the exclusion filter — there is no `@multica/mobile` workspace to exclude. The default turbo invocation runs across all real packages.
- **`packages/core/markdown/` deleted as dead code.** It was the single source of truth for `preprocessMentionShortcodes` *for mobile only* — web/desktop consumed an identical copy at `packages/ui/markdown/mentions.ts`. With mobile gone, the core copy has no callers (verified via grep — `@multica/core/markdown` is referenced nowhere outside `apps/mobile` and its own docs). Dropped the `./markdown` export from `packages/core/package.json` accordingly. The web copy at `packages/ui/markdown/mentions.ts` stays as the only definition.
- **`packages/ui/markdown/mentions.ts` docstring trimmed.** The "SYNCED COPY — keep identical to packages/core/markdown/mention-shortcodes.ts" / Package Boundary Rules block had no remaining basis; replaced with a short purpose comment.
- **"API Response Compatibility" framing in `CLAUDE.md` reworked.** The mobile-specific rationale ("the mobile app installed on a user's device is older than any backend it talks to ... CSR-only browser apps can ship a fix in minutes; an Electron build sitting on a developer's laptop cannot") collapsed once both Electron and mobile were gone. Kept the defensive-parsing rules (still good practice for any API client) but dropped the installed-app framing, the three referenced incidents (#2143, #2147, #2192 — internal to the upstream's tracker), and the trailing paragraph. The Coding Rules bullet that re-stated the mobile rationale was likewise removed.
- **`Dependency Declaration Rule` exception line dropped.** Its sole purpose was carving out `apps/mobile/` from the catalog-version rule.
- **`README.md` apps/mobile reference left intact.** Line 192 (`An iOS mobile client lives in apps/mobile/ — see its README ...`) now links to a deleted file. Issue 19 (docs-rewrite) rewrites the README from scratch — touching it here would conflict with that work. The link is broken but doesn't break any build / type / test, and the README is explicitly slated for a full rewrite.
- **`apps/docs/content/docs/mobile-app.{mdx,zh.mdx}` left intact.** Both files reference mobile; issue 04 deletes `apps/docs/` entirely, so cleaning these now would be wasted work.

### Files changed

- **Deleted directories:** `apps/mobile/` (entire workspace), `packages/core/markdown/` (dead after mobile removed).
- **Modified:** `package.json` (dropped `dev:mobile*` / `ios:mobile*` scripts, dropped `--filter='!@multica/mobile'` from turbo invocations, dropped `expo` / `react-native` / `react@19.2.0` `dependencies` block, dropped `@xmldom/xmldom` override), `turbo.json` (dropped `dev:staging` task), `.gitignore` (dropped `!apps/mobile/.env.staging` and `!apps/mobile/.env.production` exceptions), `CLAUDE.md` (dropped `apps/mobile` from architecture list, dropped "Sharing Principles" section, dropped Mobile (Expo) Commands block, dropped "Mobile-specific Rules" section, reworked Dependency Declaration Rule to drop the mobile exception, reworked API Response Compatibility framing to drop the installed-app rationale, dropped the mobile-specific Coding Rules bullet), `packages/ui/markdown/mentions.ts` (trimmed SYNCED COPY docstring), `packages/core/package.json` (dropped `./markdown` export), `pnpm-lock.yaml` (auto-regenerated; massive Expo/RN dep tree removed).

### Verification

- `pnpm install` → clean (peer-dep warning gone after re-run).
- `pnpm typecheck` → passes (5 successful workspaces: `@multica/core`, `@multica/ui`, `@multica/views`, `@multica/web`, `@multica/docs`).
- `pnpm test` → passes (`@multica/views` 821 tests across 92 files; remaining packages cached green).

### Blockers / notes for next iteration

- `apps/docs/content/docs/mobile-app.{mdx,zh.mdx}` and `README.md:192` still mention `apps/mobile`. Both surfaces are fully owned by later issues (04 deletes apps/docs, 19 rewrites README) — leaving them for those passes avoids double-touching.
- The pnpm-lock churn is large because the Expo / React Native transitive graph (~14k lines) is gone. The lockfile diff is mechanical and shouldn't require human review.
