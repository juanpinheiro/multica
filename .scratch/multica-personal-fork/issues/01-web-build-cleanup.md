# 01 — Web build cleanup: unblock Turbopack

**Status:** `done`

## Parent

`.scratch/multica-personal-fork/PRD.md`

## What to build

Remove the marketing surface that forces Next.js into webpack mode. Delete the `(landing)/*` routes, the `fumadocs-mdx` integration (config, dependencies, `.source/` artifacts, postinstall script), and the `--webpack` flag from dev/build scripts.

The root cause is documented in `apps/web/next.config.ts:74-77`: fumadocs-mdx@12 is incompatible with Next 16's Turbopack, which is why the project falls back to webpack. With fumadocs gone, Turbopack becomes available and dev rebuilds go from 2-5s to 50-200ms.

This is the first issue in the deletion effort because it gives an immediate, perceptible speed win for every subsequent issue.

## Acceptance criteria

- [x] `apps/web/app/(landing)/{about,changelog,homepage,usecases,download,contact-sales}` routes deleted
- [x] `apps/web/source.config.ts`, `apps/web/.source/`, `apps/web/content/` deleted
- [x] `fumadocs-mdx` and `fumadocs-core` removed from `apps/web/package.json` dependencies
- [x] `apps/web/package.json` scripts updated: `dev` no longer passes `--webpack`; `build` no longer prefixed with `fumadocs-mdx &&` and no `--webpack`; `typecheck` no longer prefixed with `fumadocs-mdx &&`; `postinstall` removed
- [x] `createMDX` wrapper removed from `apps/web/next.config.ts`
- [x] `/docs` rewrite removed from `apps/web/next.config.ts`
- [x] `DOCS_URL` removed from `turbo.json` `globalEnv` and `.env.example`
- [x] `dev:docs` script removed from root `package.json`
- [x] `apps/web/app/(landing)/page.tsx` deleted (root landing page) and `apps/web/app/page.tsx` shows a redirect to `/login` if no workspaces, else to the user's last/default workspace
- [x] `pnpm install` succeeds without postinstall failures
- [x] `pnpm dev:web` starts in under 5 seconds and HMR responds in under 200ms on a trivial component edit
- [x] `pnpm typecheck` and `pnpm test` pass

## Blocked by

None - can start immediately

## Comments

### Key decisions

- **Root redirect is server-side, single-line.** `apps/web/app/page.tsx` just calls `redirect("/login")`. The login page already contains the post-auth resolution logic (`resolveLoggedInDestination` — picks the user's first workspace or `/onboarding`), so bouncing through `/login` satisfies the "if no workspaces redirect to login, else to workspace" requirement without duplicating that logic at the root. Cleaner than reimplementing the workspace-lookup server-side, and a no-op for authenticated users (~1 extra redirect).
- **`DOCS_URL` was not in `.env.example`** at start (the AC line was speculative). Removed from `turbo.json` globalEnv only.
- **Deleted `sitemap.ts` and `robots.ts` from `apps/web/app/`.** Both referenced the deleted marketing routes (`/about`, `/changelog`, `/contact-sales`) and only existed for marketing-site SEO. Removing them is consistent with the personal-fork goal — no public marketing site means no sitemap.
- **Deleted unused landing image assets** (`public/images/{landing-bg.jpg,landing-hero.png,feature-bg*.jpg}`) — confirmed via grep that nothing outside the deleted `features/landing/` referenced them.
- **Login page's `/download` link left in place.** It now points at a deleted route but doesn't break typecheck or tests. Issue 09 (loopback auth + singleton user) rewrites the entire login page; touching it here would create churn against that change. Acceptable transient state.
- **Dev server confirmed running on Turbopack in 418ms** (well under the 5s AC threshold), down from webpack's ~2-5s.

### Files changed

- Created: `apps/web/app/page.tsx`
- Modified: `apps/web/next.config.ts` (drop `createMDX`, `/docs` rewrite, `docsUrl`), `apps/web/package.json` (drop fumadocs deps, drop `--webpack` / `fumadocs-mdx` prefixes, drop `postinstall`), `turbo.json` (drop `DOCS_URL` globalEnv), root `package.json` (drop `dev:docs` script), `pnpm-lock.yaml` (auto-updated)
- Deleted directories: `apps/web/app/(landing)/`, `apps/web/features/landing/`, `apps/web/content/`, `apps/web/.source/`
- Deleted files: `apps/web/source.config.ts`, `apps/web/lib/use-cases-i18n.ts`, `apps/web/lib/use-cases-source.ts`, `apps/web/app/sitemap.ts`, `apps/web/app/robots.ts`, `apps/web/public/images/{landing-bg.jpg,landing-hero.png,feature-bg.jpg,feature-bg-2.jpg,feature-bg-3.jpg,feature-bg-4.jpg}`

### Blockers / notes for next iteration

- `apps/desktop` has a **pre-existing test failure** in `scripts/package.test.mjs` (`SyntaxError: Invalid or unexpected token`) that reproduces with this branch's changes stashed — not introduced here. apps/desktop is scheduled for deletion in issue 02, which moots this. `pnpm test --filter !@multica/desktop` is green for everything else (core, ui, views, web).
- Login page still imports `paths.invitations` and the `extra` "Prefer desktop? Download" affordance. Both go away naturally — invitations in issue 05, desktop/login rewrite in issue 09.
- Root `package.json` still has `react@19.2.0`, `react-native`, `expo` direct deps and `electron` in `onlyBuiltDependencies`; mobile and desktop scripts. All cleaned up in issues 02 and 03.
- Deleted feature-bg images were not referenced by any remaining code — safe.
