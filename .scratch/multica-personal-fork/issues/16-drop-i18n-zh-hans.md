# 16 — Drop zh-Hans translations

**Status:** `done`
**Model:** `haiku`

## Parent

`.scratch/multica-personal-fork/PRD.md`

## What to build

Remove the Chinese (Simplified) translations and all infrastructure that exists to support a second locale. Only English is kept. The i18n machinery (`useT()`, namespaces, `i18next`) stays intact for forward compatibility — only the `zh-Hans` resources and the locale-switcher are removed.

## Acceptance criteria

- [ ] `packages/views/locales/zh-Hans/` directory deleted (24 JSON namespace files)
- [ ] `packages/views/locales/index.ts` reduced so `RESOURCES` only contains the `en` entry
- [ ] `packages/views/locales/parity.test.ts` simplified to validate the EN bundle is internally consistent (no second locale to compare against) — or deleted entirely if the simplified version adds no value
- [ ] `packages/views/locales/glossary.md` deleted
- [ ] `README.zh-CN.md` deleted
- [ ] `SupportedLocale` type in `@multica/core/i18n` reduced to `'en'` (single-member union); call sites simplified where the type-narrowing becomes trivial
- [ ] Locale switcher (if present in `views/settings/preferences-tab.tsx` or `views/layout/`) removed; locale is implicitly EN
- [ ] "Conventions reference" section at the top of `CLAUDE.md` (lines 5-18) trimmed: drop the Chinese voice guide bullets, drop the i18n glossary reference. The naming-convention pointer is reworked in issue 19 (docs-rewrite); for now just remove the Chinese-specific bits
- [ ] `pnpm typecheck` and `pnpm test` pass

## Blocked by

None - can start immediately

## Comments

### Key decisions

- **Simplified parity test to EN-only validation.** The old test compared zh-Hans and EN bundles for key parity. Since only EN is kept, the test was simplified to only validate that every JSON file in the `en/` directory is registered in RESOURCES. This prevents drift where a new namespace file is created but not registered.
- **All i18n type definitions updated to single-locale.** `SupportedLocale` is now a single-member union `"en"` rather than `"en" | "zh-Hans"`. Type narrowing automatically simplified at every call site since there's only one possible value. `SUPPORTED_LOCALES` is now a single-item array `["en"]`.
- **Locale adapter still present for forward compatibility.** The `useLocaleAdapter()` hook and locale persistence machinery (cookies, system preferences) are kept intact — they now trivially always resolve to `"en"`, but the infrastructure exists if locales are added again in the future.
- **i18n machinery unchanged.** `useT()`, namespaces, i18next configuration all stay intact; only the `zh-Hans` resources are gone. The infrastructure is minimal and adding a new locale in the future would only require adding JSON files and one RESOURCES entry.

### Files changed

**Deleted:**
- `packages/views/locales/zh-Hans/` (entire directory, 22 JSON namespace files)
- `packages/views/locales/glossary.md`
- `README.zh-CN.md`

**Modified:**
- `packages/core/i18n/types.ts` — `SupportedLocale` reduced to `"en"`, `SUPPORTED_LOCALES` to `["en"]`
- `packages/views/locales/index.ts` — removed all zh-Hans imports and RESOURCES entry
- `packages/views/locales/parity.test.ts` — simplified from parity comparison to EN-only consistency check
- `packages/views/locales/en/settings.json` — removed language preferences section (was only used by locale switcher)
- `packages/views/settings/components/preferences-tab.tsx` — removed language switcher UI, removed related imports (`useLocaleAdapter`, `SUPPORTED_LOCALES`, etc.), simplified component to only render theme and timezone sections
- `packages/views/settings/components/preferences-tab.test.tsx` — removed language switcher test suite, removed related mocks
- `packages/views/i18n/resources-types.ts` — updated comment to reference en/ only
- `packages/views/test/i18n.tsx` — simplified `RenderArgs.locale` type to single `"en"` option
- `packages/views/autopilots/components/autopilot-dialog-i18n.test.ts` — removed zh-Hans test section and zh-Hans JSON import
- `packages/core/i18n/browser-cookie-adapter.test.ts` — updated tests to use "en" only
- `packages/core/i18n/pick-locale.test.ts` — updated tests: zh-Hans now correctly falls back to EN
- `apps/web/lib/locale-routing.test.ts` — updated tests: zh-Hans now correctly rejected as unsupported
- `apps/web/app/layout.tsx` — removed Chinese font fallbacks, simplified HTML_LANG mapping

### Test results

- `pnpm typecheck` → 4/4 packages successful (81 tests total)
- `pnpm test` → 666 tests passed across @multica/views, @multica/web, @multica/core (all packages green)

### Blockers / notes for next iteration

None — all AC met. Issue 17 (consolidate-migrations) can proceed independently to collapse the schema and drop zh-Hans schema entries if any.
