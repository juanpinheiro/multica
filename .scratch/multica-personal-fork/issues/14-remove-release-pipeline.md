# 14 — Remove release pipeline

**Status:** `ready-for-agent`
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
