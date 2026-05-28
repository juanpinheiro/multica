# 22 — Clean up remaining stale auth references and orphan locale

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/multica-personal-fork/PRD.md`

## What to build

After issues 09, 20, and 21 remove the login flow and the PAT surface, a few cosmetic strings and one orphan i18n bundle still imply the deleted auth flow exists. Each one is small on its own, but together they mislead the user about how the fork works.

Behavior required after this slice:

- No user-facing copy mentions "sign in", "log in", "log out", "sign out", or "personal access token".
- The i18n bundle no longer carries an `auth` namespace or any orphan resource files.

Concretely:

- Delete `packages/views/locales/en/auth.json`. Remove its import and the `auth` key from `RESOURCES` in `packages/views/locales/index.ts`. Update `packages/views/i18n/resources-types.ts` if it enumerates namespaces. Run `packages/views/locales/parity.test.ts` to confirm consistency.
- Fix the "Add a computer" dialog copy in `packages/views/runtimes/components/` (whichever file owns the add-computer modal) — the line under the second code block currently reads `Opens a browser to sign in, then keeps the daemon running in the background.` and should be replaced with something accurate for the loopback model (e.g. `Keeps the daemon running in the background — it will appear here automatically.`).
- Sweep `apps/web/`, `packages/views/`, and `packages/ui/` for any other user-visible copy mentioning sign-in / log-in / OAuth / magic link / PAT. Each hit is either reworded for the loopback model or deleted.

## Acceptance criteria

- [ ] `packages/views/locales/en/auth.json` deleted
- [ ] `packages/views/locales/index.ts` no longer imports or exposes `auth`
- [ ] `packages/views/i18n/resources-types.ts` no longer lists `auth` (if it enumerates namespaces)
- [ ] `packages/views/locales/parity.test.ts` passes
- [ ] Grep for `sign in`, `sign-in`, `log in`, `log-in`, `sign out`, `log out`, `magic link`, `OAuth`, `personal access token`, `PAT` across `apps/web/`, `packages/views/`, `packages/ui/` returns no user-facing text (matches in code comments or test fixtures are OK; flag if uncertain)
- [ ] The "Add a computer" runtime dialog no longer references signing in
- [ ] `pnpm typecheck`, `pnpm test`, and `pnpm lint` pass

## Blocked by

None - can start immediately

## Comments

### Key decisions

- **`packages/views/locales/en/auth.json` deleted** and removed from `locales/index.ts` RESOURCES and `i18n/resources-types.ts` type declarations. Three test files (`mention-suggestion.test.tsx`, `search-command.test.tsx`, `preferences-tab.test.tsx`) imported `enAuth` and included it in their `TEST_RESOURCES` object — all three updated to drop the import and the `auth` key.
- **`resources-types.ts` comment updated** to replace the `t($ => $.signin.title)` example (which referenced the now-deleted `auth.json`) with a generic `t($ => $.common.field)` example.
- **`runtimes.json` connect section rewritten** for the loopback auth model:
  - `step2_hint`: "Keeps the daemon running in the background — it will appear here automatically." (was: "Opens a browser to sign in…")
  - `troubleshooting` label: "Connecting to a remote server?" (was: "Can't open a browser on that computer?")
  - `trouble_intro`, `trouble_token_hint_prefix/destination/suffix`: Updated to explain setting `MULTICA_TOKEN` env var on the server (was: pointing at the deleted Settings → Tokens tab).
- **`connect-remote-dialog.tsx` TOKEN_CMD updated**: Replaced the old flow (`multica config set app_url https://api.multica.ai; multica login --token <YOUR_TOKEN>; multica daemon start`) with the new loopback-auth equivalent (`export MULTICA_TOKEN=<token>; multica config set server_url http://<host>:8080; multica daemon start`).
- **Dead locale strings removed**: `layout.json` sidebar `log_out`, `workspace.json` new_page `log_out` and no_access `sign_in_different` — none were referenced by any component (confirmed by grep), safe to delete.
- **`capability-banner.tsx` `not_authenticated` case**: Changed "Sign in to edit this ${noun}." → "View only." This state is transient (resolves to the singleton user once data loads) but should not display a sign-in prompt even briefly.
- **Remaining grep hits are acceptable**: `packages/views/common/task-transcript/redact.ts` has "GitLab personal access tokens" in a code comment (AC explicitly allows comments); `github-tab.tsx` hits are false positives from case-insensitive `OAuth` matching `coAuthor`.

### Files changed

- **Deleted:** `packages/views/locales/en/auth.json`
- **Modified:** `packages/views/locales/index.ts` (removed `enAuth` import and `auth` RESOURCES entry)
- **Modified:** `packages/views/i18n/resources-types.ts` (removed auth type import and declaration, updated example comment)
- **Modified:** `packages/views/locales/en/runtimes.json` (connect dialog strings: step2_hint, troubleshooting, trouble_intro, trouble_token_hint_*)
- **Modified:** `packages/views/runtimes/components/connect-remote-dialog.tsx` (TOKEN_CMD constant updated)
- **Modified:** `packages/views/locales/en/layout.json` (removed dead `sidebar.log_out`)
- **Modified:** `packages/views/locales/en/workspace.json` (removed dead `new_page.log_out` and `no_access.sign_in_different`)
- **Modified:** `packages/ui/components/common/capability-banner.tsx` (not_authenticated → "View only.")
- **Modified:** `packages/views/editor/extensions/mention-suggestion.test.tsx` (removed enAuth import and TEST_RESOURCES entry)
- **Modified:** `packages/views/search/search-command.test.tsx` (removed enAuth import and TEST_RESOURCES entry)
- **Modified:** `packages/views/settings/components/preferences-tab.test.tsx` (removed enAuth import and TEST_RESOURCES entry)

### Verification

- `pnpm test` → 6/6 packages pass (636+ tests)
- `pnpm typecheck` → 4/4 packages pass
