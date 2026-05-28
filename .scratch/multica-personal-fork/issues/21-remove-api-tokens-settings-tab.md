# 21 — Remove API Tokens tab from Settings

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/multica-personal-fork/PRD.md`

## What to build

Issue 09 deleted `handler/personal_access_token.go` and the `auth.PATCache`, removing the entire personal-access-token surface from the backend. The Settings UI in `packages/views/settings/components/settings-page.tsx` still renders an "API Tokens" tab (`TokensTab`), still registered as one of the account tab keys, and the page still ships the tab's i18n strings. Hitting "Create" in that tab fires a request to `/api/tokens` which returns 404 — the tab is visibly broken.

Behavior required after this slice:

- The Settings page exposes only the tabs whose backends still exist: Profile, Preferences, Notifications, General, Repositories, GitHub, Integrations, Labs.
- No code under `packages/views/`, `packages/core/`, or `apps/web/` references PAT creation, listing, revocation, or the `/api/tokens` endpoint.

The work touches:

- Remove the `TokensTab` import, the `tokens` key from `ACCOUNT_TAB_KEYS` (and any other tab registry), and the `<TabsContent value="tokens">` block in `settings-page.tsx`.
- Delete `packages/views/settings/components/tokens-tab.tsx` and any test file for it.
- Delete the `api.tokens.*` methods in `packages/core/api/` (and the corresponding schema in `packages/core/api/schema.ts` if present).
- Remove the `tokens` namespace section from `packages/views/locales/en/settings.json`.
- Remove any TanStack Query keys / hooks that reference tokens (e.g. `packages/core/tokens/queries.ts` if it exists).

## Acceptance criteria

- [ ] `/personal/settings` (or any workspace slug) renders 8 tabs: Profile, Preferences, Notifications, General, Repositories, GitHub, Integrations, Labs
- [ ] `packages/views/settings/components/tokens-tab.tsx` deleted; no remaining imports of it
- [ ] No code under `packages/views/`, `packages/core/`, or `apps/web/` matches the patterns `api/tokens`, `TokensTab`, `personal_access_token`, `PATCache`, or `useTokens` (verified by grep)
- [ ] `packages/views/locales/en/settings.json` no longer contains a `tokens` namespace
- [ ] `pnpm typecheck`, `pnpm test`, and `pnpm lint` pass
- [ ] `make check` passes

## Blocked by

None - can start immediately

## Comments

### Key decisions

- **No `packages/core/tokens/` directory existed.** The AC mentioned checking for a `tokens/queries.ts` file; no such directory was present. All PAT state was in-component (`useState` inside `tokens-tab.tsx`), so no TanStack Query keys or Zustand store needed to be removed.
- **PAT type definitions removed from `packages/core/types/api.ts`.** `PersonalAccessToken`, `CreatePersonalAccessTokenRequest`, and `CreatePersonalAccessTokenResponse` had no callers outside `tokens-tab.tsx` and `client.ts` — both deleted or cleaned in this slice.
- **`Key` lucide icon import removed from `settings-page.tsx`.** The icon was only used for `tokens: Key` in `ACCOUNT_TAB_ICONS`; nothing else referenced it.
- **`"tokens": "API Tokens"` entry removed from `page.tabs` in `settings.json`.** The type is inferred from the JSON, so removing the key here also drops it from the `I18nResources` type automatically.
- **No test file for `tokens-tab.tsx` existed.** The AC mentioned "and any test file for it" — confirmed by search; no `tokens-tab.test.tsx` was present.

### Files changed

- **Deleted:** `packages/views/settings/components/tokens-tab.tsx`
- **Modified:** `packages/views/settings/components/settings-page.tsx` — removed `Key` icon import, `TokensTab` import, `"tokens"` from `ACCOUNT_TAB_KEYS`, `tokens: Key` from `ACCOUNT_TAB_ICONS`, `<TabsContent value="tokens">` render
- **Modified:** `packages/core/api/client.ts` — removed `PersonalAccessToken` / `CreatePersonalAccessTokenRequest` / `CreatePersonalAccessTokenResponse` type imports and `listPersonalAccessTokens` / `createPersonalAccessToken` / `revokePersonalAccessToken` methods
- **Modified:** `packages/core/types/api.ts` — removed `PersonalAccessToken`, `CreatePersonalAccessTokenRequest`, `CreatePersonalAccessTokenResponse` interfaces
- **Modified:** `packages/views/locales/en/settings.json` — removed `"tokens"` key from `page.tabs` and the entire `tokens` namespace section

### Verification

- `pnpm typecheck` → 4/4 packages green
- `pnpm test` → 666 tests / 81 files pass across all packages
