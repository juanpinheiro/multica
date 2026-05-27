# 14 — Remove release pipeline

**Status:** `done`
**Model:** `haiku`

## Parent

`.scratch/multica-personal-fork/PRD.md`

## What to build

Remove the GoReleaser-based CLI release pipeline and all its supporting assets. The user builds the CLI from their own checkout; there's no Homebrew tap, no install script, no GitHub release Action.

## Acceptance criteria

- [ ] `.goreleaser.yml` deleted
- [ ] `.github/workflows/release.yml` deleted
- [ ] `scripts/install.sh` and `scripts/install.ps1` deleted
- [ ] "CLI Release" section deleted from `CLAUDE.md`
- [ ] Homebrew tap and install-script references in `README.md` deleted (README is rewritten in 19, but a quick pass here is fine)
- [ ] CLI `multica update` command in `internal/cli/update.go` reviewed: keep a minimal "build from source" hint or delete the auto-update path entirely (Homebrew/install-script handling no longer applies)
- [ ] No build pipeline configuration remains that references GoReleaser, Homebrew, or third-party install tooling

## Blocked by

None - can start immediately

## Comments

### Key decisions

- **Build-from-source is the only installation method.** Users clone the repo and run `make build` or `go build` directly. No Homebrew tap, no GitHub releases, no install scripts.
- **Updated all active code and documentation.** Installation references in `CLI_AND_DAEMON.md` and `CLI_INSTALL.md` were rewritten to build-from-source. The `connect-remote-dialog.tsx` component that shows CLI installation instructions to users was updated with the new build command.
- **Makefile Homebrew references replaced.** The `selfhost` and `selfhost-build` targets that previously recommended `brew install multica-ai/tap/multica` now show the build-from-source command instead.
- **Stale references in docs-to-be-deleted are left in place.** References in `SELF_HOSTING.md`, `SELF_HOSTING_AI.md`, and `README.zh-CN.md` are scheduled for deletion in issues 15 and 16, so cleaning them here would create churn. Issue 14's AC correctly focuses on active code and documentation.

### Files changed

**Deleted:**
- `.goreleaser.yml` (GoReleaser configuration)
- `.github/workflows/release.yml` (GitHub Actions release workflow)
- `scripts/install.sh` and `scripts/install.ps1` (install scripts)
- `scripts/install.test.sh` (tests for deleted install.sh)

**Modified:**
- `CLI_AND_DAEMON.md` — Removed "Homebrew" installation section, consolidated "Build from Source" as the only method with both `make build` and direct `go build` instructions
- `CLI_INSTALL.md` — Completely rewritten from multi-option install guide to single build-from-source method with prerequisites and step-by-step instructions
- `Makefile` — Updated `selfhost` and `selfhost-build` targets to replace `brew install multica-ai/tap/multica` with `cd server && go build ... && multica setup` instructions
- `.vercelignore` — Removed stale `.goreleaser.yml` reference
- `packages/views/runtimes/components/connect-remote-dialog.tsx` — Updated `INSTALL_CMD` to reference build-from-source command instead of deleted install.sh script

### Verification

- `pnpm typecheck` — 4/4 packages pass
- `pnpm test` — 718 tests / 81 files pass across `@multica/core`, `@multica/ui`, `@multica/views`, `@multica/web`
- All acceptance criteria met:
  - ✓ `.goreleaser.yml` deleted
  - ✓ `.github/workflows/release.yml` deleted
  - ✓ `scripts/install.sh` and `scripts/install.ps1` deleted
  - ✓ No "CLI Release" section in `CLAUDE.md`
  - ✓ No Homebrew references in `README.md`
  - ✓ No `multica update` auto-update command found (doesn't exist)
  - ✓ No build pipeline configuration references to GoReleaser, Homebrew, or third-party install tooling in active code

### Blockers / notes for next iteration

- Documentation files `SELF_HOSTING.md`, `SELF_HOSTING_AI.md`, `README.zh-CN.md` still contain references to install scripts and Homebrew, but these are scheduled for deletion in issues 15 (remove self-hosting docs) and 16 (drop i18n). Cleaning them here would conflict with those issues.
- The changes to `connect-remote-dialog.tsx` require users to have Go, Git, and a build environment available. This is acceptable for a personal fork where the developer maintains their own checkout.
