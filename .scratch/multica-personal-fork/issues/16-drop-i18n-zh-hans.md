# 16 — Drop zh-Hans translations

**Status:** `ready-for-agent`
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
