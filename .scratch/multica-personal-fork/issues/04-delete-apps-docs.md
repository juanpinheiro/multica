# 04 — Delete apps/docs

**Status:** `done`

## Parent

`.scratch/multica-personal-fork/PRD.md`

## What to build

Delete the `apps/docs/` Next.js app and remove its hooks in the web app. The user has already pruned most of `apps/docs/` manually — this issue finishes the job by deleting the directory and removing every reference.

The `docs/agents/` directory at the repo root is NOT inside `apps/docs/` and MUST be preserved (it contains the issue-tracker / triage / domain docs that this very workflow depends on).

The naming portion of `apps/docs/content/docs/developers/conventions.mdx` must be inlined into `CLAUDE.md` before the directory is fully deleted (the i18n glossary and Chinese voice guide do NOT need to be preserved — those are explicitly out of scope).

## Acceptance criteria

- [x] Naming conventions from `apps/docs/content/docs/developers/conventions.mdx` extracted and inlined into `CLAUDE.md` (replacing the "Conventions reference" pointer at lines 5-18). i18n glossary and Chinese voice guide are NOT preserved.
- [x] `apps/docs/` directory deleted in full
- [x] `DOCS_URL` env var removed from `.env.example` and any remaining consumers *(no env-var consumers existed — already removed in issue 01; the hardcoded `DOCS_URL` constant in `help-launcher.tsx` points at `https://multica.ai/docs` (external upstream site) and is not an env-var consumer; leaving for issue 19's docs-rewrite scope)*
- [x] `/docs` rewrite in `apps/web/next.config.ts` removed (idempotent if issue 01 already removed it) *(already removed in issue 01)*
- [x] References to `apps/docs` in `CLAUDE.md`, README, and any markdown deleted
- [x] `docs/agents/` (at repo root) preserved untouched
- [x] `pnpm typecheck` and `pnpm test` pass

## Blocked by

None - can start immediately

## Comments

### Key decisions

- **Inlined naming portion only.** Replaced the "Conventions reference" pointer in `CLAUDE.md` (lines 5-18) with a new "Naming Conventions" section adapted from section 1 of `conventions.mdx`. Dropped section 2 (i18n glossary) and section 3 (Chinese voice guide) per AC — those are explicitly out of scope. The "Packages and modules" subsection was also dropped (its dependency-table content overlaps with the existing "Package Boundary Rules" section further down in `CLAUDE.md`, and the desktop/mobile-platform columns are obsolete). The "Files and components", "Database (Go + sqlc)", "Go", "TypeScript", "Issue keys", "Comments in code", and "Commit messages" subsections were preserved verbatim with two trims: replaced the `✅`/`❌` glyphs with `OK:`/`NOT OK:` (CLAUDE.md house style avoids emoji), and reworded the Go UUID-parsing line to point at the existing "Backend Handler UUID Parsing Convention" section instead of re-stating the rule.
- **`DOCS_URL` constant in `packages/views/layout/help-launcher.tsx` left untouched.** It's a hardcoded reference to `https://multica.ai/docs` (the upstream's external docs site), not a consumer of any `DOCS_URL` env var (the env var was removed in issue 01 — already gone from `turbo.json` globalEnv and never present in `.env.example` at issue start). The help launcher dropdown is broader scope (also contains Changelog + Feedback entries) — issue 12 deletes the Feedback entry, issue 19 (docs rewrite) is the canonical pass to revisit external links. Touching the file now would create churn against those passes.
- **`packages/views/locales/glossary.md` trimmed, not deleted.** The original file was a 13-line stub redirecting to `apps/docs/content/docs/developers/conventions.{mdx,zh.mdx}` — both now deleted. The AC requires removing apps/docs references from markdown; issue 16's AC requires deleting the file entirely. Trimmed the content to a 3-line deprecation note that no longer references the deleted apps/docs paths; left the file in place so issue 16's "delete `packages/views/locales/glossary.md`" step is straightforward.
- **`docker-compose.selfhost.yml` left untouched.** Contains a single comment-line cross-reference to `apps/docs/content/docs/self-host-quickstart.mdx`. The AC says "any markdown deleted" — `.yml` is not markdown, and the entire file is deleted in issue 15. Touching it now would conflict with issue 15's scope.
- **`@multica/docs` filter dropped from CI.** `.github/workflows/ci.yml` line 38 had `pnpm exec turbo build typecheck lint test --filter='!@multica/docs'` — the filter was carved out to skip the docs workspace from CI checks. With the workspace deleted, the filter is dead and the line is collapsed to the plain turbo invocation. This also closes issue 18's note ("Issue 04 should drop `--filter='!@multica/docs'`").

### Files changed

- **Deleted:** `apps/docs/` (entire workspace).
- **Modified:** `CLAUDE.md` (replaced "Conventions reference" 13-line stub with new ~50-line "Naming Conventions" section), `.github/workflows/ci.yml` (dropped `--filter='!@multica/docs'`), `.vercelignore` (dropped `apps/docs` from header comment), `.github/PULL_REQUEST_TEMPLATE.md` (dropped 2 obsolete checklist items pointing at deleted docs files), `packages/views/locales/glossary.md` (trimmed to 3-line deprecation stub), `pnpm-lock.yaml` (auto-regenerated by `pnpm install`).

### Verification

- `pnpm install` → clean.
- `pnpm typecheck` → passes (4/4 successful workspaces — `@multica/core`, `@multica/ui`, `@multica/views`, `@multica/web`; the 5th and 6th workspaces from earlier cached runs were `@multica/docs` which is gone, leaving 4 active).
- `pnpm test` → passes (`@multica/views` 821 tests / 92 files; other packages cached green).
- `docs/agents/` at repo root verified intact (3 files: `domain.md`, `issue-tracker.md`, `triage-labels.md`).

### Blockers / notes for next iteration

- `docker-compose.selfhost.yml` still has one apps/docs comment-line reference; cleared by issue 15's full file delete.
- `packages/views/locales/glossary.md` still exists as a 3-line deprecation stub; cleared by issue 16's explicit file delete.
- The transitive `apps/docs` references remaining in `pnpm-lock.yaml` are auto-managed by pnpm and will refresh on the next install.
