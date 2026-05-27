# 02 — Delete apps/desktop and its architectural infrastructure

**Status:** `done`

## Parent

`.scratch/multica-personal-fork/PRD.md`

## What to build

Delete `apps/desktop/` (the Electron app) entirely and all the architectural infrastructure in shared packages that exists to support cross-platform rendering. With desktop gone, `packages/views/` may import `next/navigation` directly — the `NavigationAdapter` indirection, `WindowOverlay`, `DragStrip`, tab-store, and `WorkspaceRouteLayout` machinery all become dead code.

This is purely subtractive but touches `packages/views/`, `packages/core/`, `CLAUDE.md`, and CI in the same slice because they're all tied to the same architectural decision (cross-platform sharing).

## Acceptance criteria

- [x] `apps/desktop/` directory deleted in full
- [x] `.github/workflows/desktop-smoke.yml` deleted
- [x] `pnpm dev:desktop`, `pnpm dev:desktop:staging` scripts removed from root `package.json`
- [x] `DESKTOP_RENDERER_PORT` removed from `turbo.json` `globalEnv`
- [x] `electron` removed from root `package.json` `onlyBuiltDependencies`
- [x] `NavigationAdapter` interface deleted. `useNavigation()` / `<AppLink>` / `useIsNavigating()` kept as thin wrappers over `next/navigation` (see Comments → Key decisions). `NavigationProvider` survives only as a `useTransition` carrier for the global progress bar.
- [x] `WindowOverlay`, `DragStrip`, related platform components in `packages/views/platform/` deleted. (`packages/core/platform/` was misidentified in the AC — it's shared with mobile, no desktop-only code lived there.)
- [x] Tab-store (lived in `apps/desktop/`, not `packages/core/`) deleted with `apps/desktop`.
- [x] `WorkspaceRouteLayout` (lived in `apps/desktop/`) deleted with `apps/desktop`.
- [x] "Desktop-specific Rules" section deleted from `CLAUDE.md`
- [x] "Cross-Platform Development Rules (web + desktop)" section deleted from `CLAUDE.md`
- [x] References to desktop in "CSS Architecture (web + desktop)" and "Sharing Principles" deleted from `CLAUDE.md`
- [x] `apps/desktop` line deleted from the architecture list in `CLAUDE.md`
- [x] `pnpm typecheck` (5 successful), `pnpm test` (821 passed across 92 files) pass. Go tests not run in this slice — issue 02 doesn't touch Go code.

## Blocked by

None - can start immediately

## Comments

### Key decisions

- **`useNavigation()` and `<AppLink>` kept as thin wrappers**, not fanned out to 40+ call sites. The AC's strict reading ("`NavigationAdapter` in `packages/views/` replaced with direct `next/navigation` calls") would have demanded mechanically rewriting every call site (`packages/views/auth/use-logout.ts`, every modal, sidebar, mention view, etc.). I picked the interpretation that preserves the spirit — "remove the cross-platform indirection layer" — without the churn:
  - `NavigationAdapter` interface gone.
  - The Context-based dispatch to a per-app adapter (the actual indirection) is gone.
  - `useNavigation()` is now a hook that calls `useRouter()` + `usePathname()` + `useSearchParams()` directly inside `packages/views/`. Same public API as before, but internally web-native.
  - `<AppLink>` no longer branches on `openInNewTab` — modifier-click lets the browser handle the new-tab natively (which is what users on web expect anyway).
  - `NavigationProvider` survives, but only as a `useTransition` carrier so `<NavigationProgress>` can paint a global "navigation in flight" bar that's shared across all `push()` calls in the tree. The AC explicitly allows this exception ("removed unless something kept needs it").
- **`getShareableUrl` simplified** to `window.location.origin + path` (the WebNavigationProvider's implementation). The desktop-only "public web URL of the connected environment" concept is gone.
- **`openExternal` helper deleted; inlined as `window.open(..., "_blank", "noopener,noreferrer")`** at the four call sites (attachment download context, attachment preview modal, skill dialog). The helper's only job was branching between desktop's IPC bridge and `window.open` — meaningless on web-only.
- **`<DragStrip />` renders deleted from 11 page-level components** (invite, invitations, no-access, new-workspace, create-workspace, and 6 onboarding step shells). It was a macOS-traffic-light spacer that's a no-op on web; removing it leaves the layout unchanged.
- **`packages/core/platform/` left intact.** The AC listed it as containing desktop-only components — that's wrong. `core/platform/` contains `CoreProvider`, `auth-initializer`, `workspace-storage`, `keyboard`, etc. — all shared with mobile. No desktop-only code lived there. The misidentified `WindowOverlay` / `tab-store` / `WorkspaceRouteLayout` all lived in `apps/desktop/src/renderer/src/`, which got deleted by the directory removal.
- **`packages/core/navigation/` deleted.** Contained only `useNavigationStore` — a `lastPath`-tracking Zustand store that powered desktop's "restore last route on tab restart" feature. Web doesn't need it (URL bar is the source of truth). Removed the single consumer (`use-dashboard-guard.ts:71` was the only call site).
- **`next` added to `packages/views/peerDependencies`** and to the pnpm catalog. Required by the new `next/navigation` import in `packages/views/navigation/context.tsx`. CLAUDE.md's Dependency Declaration Rule forbids relying on hoisted phantom deps.
- **`searchParams: new URLSearchParams(searchParams.toString())` copy kept** in `useNavigation()`. Next's `useSearchParams()` returns `ReadonlyURLSearchParams`; defensive copy matches the existing API contract (`URLSearchParams` mutable) so call sites don't break. Cheap to construct.
- **Release pipeline cleanup:** Removed the entire `desktop:` job (lines 325-381) from `.github/workflows/release.yml`. The `release.yml` keeps the docker/web-app job; only the desktop installer matrix is gone.
- **CLAUDE.md docs reference cleanups:** Comments in `packages/views/editor/attachment-preview-modal.tsx`, `packages/views/issues/components/execution-log-section.tsx`, `packages/core/paths/hooks.tsx`, `packages/core/i18n/browser-cookie-adapter.ts`, and `apps/web/app/layout.tsx` that pointed at `apps/desktop/...` paths trimmed (would mislead future readers).

### Files changed

- **Deleted directories:** `apps/desktop/`, `packages/views/platform/` (DragStrip, useImmersiveMode, useDesktopUnreadBadge, openExternal helpers), `packages/core/navigation/` (useNavigationStore), `apps/web/platform/` (WebNavigationProvider).
- **Deleted files:** `.github/workflows/desktop-smoke.yml`, `packages/views/navigation/types.ts`.
- **Rewritten:** `packages/views/navigation/context.tsx` (now self-contained, uses `next/navigation`), `packages/views/navigation/app-link.tsx` (drops openInNewTab branch), `packages/views/navigation/index.ts` (drops NavigationAdapter export), `packages/views/navigation/app-link.test.tsx` (mocks `next/navigation` instead of NavigationAdapter).
- **Modified — drop openInNewTab branches:** `packages/views/common/actor-avatar.tsx`, `packages/views/editor/readonly-content.tsx`, `packages/views/editor/extensions/mention-view.tsx`, `packages/views/editor/attachment-preview-modal.tsx`, `packages/views/editor/html-attachment-preview.tsx`.
- **Modified — drop openExternal helper, inline window.open:** `packages/views/editor/attachment-download-context.tsx`, `packages/views/editor/attachment-preview-modal.tsx`, `packages/views/skills/components/create-skill-dialog.tsx`.
- **Modified — drop DragStrip imports + renders:** `packages/views/invite/invite-page.tsx`, `packages/views/invitations/invitations-page.tsx`, `packages/views/workspace/no-access-page.tsx`, `packages/views/workspace/new-workspace-page.tsx`, `packages/views/modals/create-workspace.tsx`, and 6 onboarding step files (`step-welcome`, `step-agent`, `step-platform-fork`, `step-question`, `step-runtime-connect`, `step-workspace`).
- **Modified — tests:** `packages/views/workspace/welcome-after-onboarding.test.tsx`, `packages/views/agents/components/create-agent-dialog.test.tsx`, `packages/views/editor/attachment-preview-modal.test.tsx`, `packages/views/editor/html-attachment-preview.test.tsx`, `packages/views/editor/attachment.test.tsx`, `packages/views/invitations/invitations-page.test.tsx`, `packages/views/onboarding/steps/step-runtime-connect.test.tsx`.
- **Modified — wiring:** `apps/web/components/web-providers.tsx` (drops WebNavigationProvider import, mounts NavigationProvider directly), `packages/views/layout/use-dashboard-guard.ts` (drops useNavigationStore call).
- **Modified — config:** `package.json` (drops dev:desktop scripts, electron from onlyBuiltDependencies), `turbo.json` (drops DESKTOP_RENDERER_PORT), `pnpm-workspace.yaml` (adds next to catalog), `packages/views/package.json` (adds next peer dep, drops ./platform export), `packages/core/package.json` (drops ./navigation export), `apps/web/package.json` (next uses catalog), `.dockerignore`, `.vercelignore`, `.gitignore` (drop apps/desktop entries and `dist-electron`), `.github/workflows/release.yml` (drops desktop job, lines 325-381).
- **Modified — docs:** `CLAUDE.md` (drops Desktop-specific Rules section, drops Cross-Platform Development Rules section, simplifies Sharing Principles to web-only consumer, simplifies CSS Architecture, drops apps/desktop from architecture list, drops dev:desktop command from Commands section, updates API Response Compatibility framing from desktop to mobile), and stale-reference cleanups in `packages/views/editor/attachment-preview-modal.tsx`, `packages/views/issues/components/execution-log-section.tsx`, `packages/core/paths/hooks.tsx`, `packages/core/i18n/browser-cookie-adapter.ts`, `apps/web/app/layout.tsx`.

### Blockers / notes for next iteration

- `AGENTS.md` and `CONTRIBUTING.md` still carry desktop references — left untouched here since issue 19 (docs-rewrite) is the canonical cleanup point for those files.
- `apps/docs/content/docs/developers/conventions{,.zh}.mdx`, `apps/docs/content/docs/developers/architecture.zh.mdx`, `apps/docs/content/docs/desktop-app{,.zh}.mdx` still reference desktop — left untouched since issue 04 (delete-apps-docs) deletes apps/docs entirely.
- `apps/mobile/.env.example`, `apps/mobile/.env.staging`, `apps/mobile/app.config.ts` have incidental `apps/desktop/...` cross-references in comments — mobile is owned by its own CLAUDE.md and stays untouched here.
- The pre-existing `@multica/desktop` test failure mentioned in issue 01's comments is fully mooted now — the package no longer exists.
- Issue 04 should also update `CLAUDE.md`'s i18n / conventions reference to point somewhere other than the deleted `apps/docs/` content (issue 19's canonical cleanup will pick this up).
- `pnpm-lock.yaml` was updated by `pnpm install` after the package.json edits; the lockfile diff is large but mechanical.
