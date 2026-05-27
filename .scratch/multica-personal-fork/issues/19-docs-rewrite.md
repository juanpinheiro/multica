# 19 — Docs rewrite (README, CLAUDE.md, reserved slugs, assets)

**Status:** `ready-for-agent`
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
