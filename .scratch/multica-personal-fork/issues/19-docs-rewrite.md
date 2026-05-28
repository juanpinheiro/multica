# 19 — Docs rewrite (README, CLAUDE.md, reserved slugs, assets)

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/multica-personal-fork/PRD.md`

## What to build

Rewrite the user-facing documentation to reflect the personal-fork reality. README becomes a short orientation. CLAUDE.md loses every rule that exists only to support deleted platforms or features. `reserved_slugs.json` is reduced to only the slugs that correspond to routes that actually exist. Marketing assets are deleted.

This is the last issue in the deletion effort because it reflects the post-cut state of every other issue.

## Acceptance criteria

### README.md

- [ ] `README.md` rewritten as a short personal-fork orientation:
  - Name + one-sentence description
  - "This is a personal fork of Multica. Removed: ..." (concise bullet list)
  - How to run: `make dev`, then visit `http://localhost:3000`
  - Where to look for everything else: `CLAUDE.md`
  - No badges, no screenshots, no marketing language, no "Why Multica?" section
- [ ] `README.zh-CN.md` already deleted in issue 16; verify
- [ ] No leftover references to Homebrew tap, install scripts, self-hosting docs, contact sales, or cloud waitlist

### CLAUDE.md

- [ ] "Mobile-specific Rules" section deleted (already gone after 03 — verify)
- [ ] "Desktop-specific Rules" section deleted (already gone after 02 — verify)
- [ ] "Sharing Principles" section reduced to a single line stating that web is the only consumer of shared packages (or deleted entirely)
- [ ] "Cross-Platform Development Rules (web + desktop)" section deleted (already gone after 02 — verify)
- [ ] "CSS Architecture (web + desktop)" section's reference to desktop deleted; section simplified to "CSS Architecture"
- [ ] "CLI Release" section deleted (already gone after 14 — verify)
- [ ] "Multi-tenancy" section simplified: single-user reality, workspaces still exist as organizational folders
- [ ] "Project Context" section simplified: no longer "built for 2-10 person AI-native teams"; personal fork for solo use
- [ ] `apps/desktop` and `apps/mobile` lines deleted from the architecture list
- [ ] Naming conventions (from former `apps/docs/content/docs/developers/conventions.mdx`) inlined near the top, replacing the "Conventions reference" pointer at lines 5-18. Keep only the naming portion; the i18n glossary and Chinese voice guide are NOT preserved

### reserved_slugs.json

- [ ] `server/internal/handler/reserved_slugs.json` reduced to:
  - Technical slugs that exist as routes: `api`, `auth`, `ws`, `uploads`, `health`, `healthz`, `readyz`
  - Workspace-level slugs that exist as routes under `apps/web/app/[workspaceSlug]/(dashboard)/`: `agents`, `autopilots`, `inbox`, `issues`, `members`, `my-issues`, `projects`, `runtimes`, `settings`, `skills`, `squads`, `usage`
  - Any other slug owned by a top-level route in `apps/web/app/`
- [ ] `pnpm generate:reserved-slugs` run; `packages/core/paths/reserved-slugs.ts` regenerated and committed
- [ ] CI re-runs the generator and confirms no drift

### Marketing assets

- [ ] `docs/assets/` deleted (banner, logos, hero screenshot)
- [ ] Any logo asset still referenced by the web app (login screen, header) moved to `apps/web/public/` and the import path updated

### Other docs

- [ ] `CONTRIBUTING.md` reviewed; stale references to multi-user / desktop / mobile / Homebrew / contact-sales removed
- [ ] `AGENTS.md` reviewed and trimmed similarly
- [ ] `docs/agents/` left untouched (the issue-tracker, triage-labels, and domain docs that this very workflow depends on)

### Verification

- [ ] `pnpm typecheck` and `pnpm test` pass
- [ ] A reader who lands on the repo with only `README.md` + `CLAUDE.md` can run `make dev` and reach a working dashboard without consulting any external resource

## Blocked by

- 02-delete-apps-desktop
- 03-delete-apps-mobile
- 04-delete-apps-docs
- 05-remove-invitations
- 06-remove-members-and-roles
- 07-remove-email-and-notification-email-side
- 08-remove-onboarding
- 09-loopback-auth-and-singleton-user
- 10-remove-cloud-runtime-fleet
- 11-remove-analytics
- 12-remove-contact-sales-and-feedback
- 13-remove-cloudfront-realtime-metrics-and-redis
- 14-remove-release-pipeline
- 15-remove-self-hosting-docs-and-compose
- 16-drop-i18n-zh-hans

## Comments

### Key decisions

- **`GLOBAL_PREFIXES` in `packages/core/paths/paths.ts` reduced to `["/workspaces/"]`.** The old list included `/logout` and `/signup`, both of which no longer exist as routes after issue 09 (loopback auth deleted the auth route group). Removing them from `GLOBAL_PREFIXES` and the corresponding consistency/paths tests is the correct cleanup rather than adding them to the reserved slugs list.
- **`reserved_slugs.json` reduced from ~90 slugs across 9 groups to 22 slugs across 3 groups.** Removed all marketing/brand/billing/platform-safety slugs that only applied to the SaaS product. Kept only: (1) pre-workspace frontend routes (`login`, `auth`, `workspaces`), (2) backend API/health routes (`api`, `ws`, `uploads`, `health`, `healthz`, `readyz`), and (3) workspace route segments (`agents`, `attachments`, `autopilots`, `inbox`, `issues`, `members`, `my-issues`, `projects`, `runtimes`, `settings`, `skills`, `squads`, `usage`). `attachments` added since `[workspaceSlug]/attachments/[id]/` is an active route.
- **`create-workspace-form.test.tsx` reserved-slug test updated from `"admin"` to `"api"`.** `admin` was removed from the reserved list; `api` is still reserved and equally valid for testing the form's validation behavior.
- **`docs/assets/` didn't exist** — the README referenced images there but the directory was never committed. No files to delete; the rewritten README simply omits those references.
- **CONTRIBUTING.md "Full-Stack Isolated Testing" section rewritten** to reflect loopback auth: the multi-step magic-link + PAT creation flow is gone; workspace creation now works without any `Authorization` header.
- **CONTRIBUTING.md "Desktop App Local Testing" section deleted** — desktop was removed in issue 02.
- **AGENTS.md package boundaries updated** — removed `apps/desktop/`, removed the `NavigationAdapter` constraint from `packages/views/` (which can now import `next/*` directly per issue 02).
- **CLAUDE.md stale items cleaned**: removed E2E from testing infrastructure table, commands, and pre-push check descriptions; removed E2E test example block; updated single-TS-test examples to non-deleted test files; updated "Project Context" to reflect single-user personal fork; updated "Multi-tenancy" to reflect singleton implicit user; updated `make check` description.
- **README.md fully rewritten** as a short personal-fork orientation with: name + one-sentence description, removed features bullet list, run instructions, architecture table, pointer to CLAUDE.md.

### Files changed

- `README.md` — complete rewrite (personal-fork orientation)
- `CLAUDE.md` — targeted edits: Project Context, Commands (E2E removed, test examples updated), Testing Rules (E2E row and section removed), Minimum Pre-Push Checks (E2E/Playwright removed), Multi-tenancy (singleton user)
- `server/internal/handler/reserved_slugs.json` — reduced from ~90 slugs to 22
- `packages/core/paths/reserved-slugs.ts` — regenerated by `pnpm generate:reserved-slugs`
- `packages/core/paths/paths.ts` — `GLOBAL_PREFIXES` reduced to `["/workspaces/"]`
- `packages/core/paths/paths.test.ts` — updated `isGlobalPath` test to match new prefixes
- `packages/core/paths/consistency.test.ts` — updated `globalPrefixes` list to match
- `packages/views/workspace/create-workspace-form.test.tsx` — reserved slug test updated from `admin` to `api`
- `CONTRIBUTING.md` — removed E2E from testing, rewrote Full-Stack Isolated Testing (loopback auth), removed Desktop App Local Testing, updated Local Daemon section
- `AGENTS.md` — removed `apps/desktop/` from architecture, updated `packages/views/` package boundary rule

### Blockers / notes for next iteration

None — all acceptance criteria met. All 6/6 TS test packages pass (666 tests) and 4/4 typecheck. This is the last issue in the multica-personal-fork effort.
