# Issue 17: CLI comment listing — roots-only + drop preamble

**Status:** `done`
**Model:** `sonnet`

## Parent

PRD 1 — Upstream Sync (`.scratch/upstream-sync/PRD.md`).

## What to build

Two CLI comment-listing improvements ported from upstream: default the comment list to roots-only output (thread-rooted, not flattened), and remove the "Showing N comments." stderr preamble. Together they produce clean output for both human use and the `multica mcp` server that wraps the CLI.

## Acceptance criteria

- [x] `multica issue comment list` defaults to roots-only output.
- [x] The "Showing N comments." stderr preamble is removed; stderr is not polluted for piped/scripted use.
- [x] CLI tests cover the roots-only default and the absence of the preamble.

## Blocked by

None — can start immediately.

## Comments

### Key decisions

- **Client-side filter for roots-only.** When neither `--thread` nor `--recent` is set (the default mode), `runIssueCommentList` filters the server response to comments where `parent_id` is absent or empty. This requires no server changes and no new SQL query — the server already returns all comments; the CLI just drops replies. `--thread` bypasses the filter (user explicitly asked for a thread); `--recent` bypasses it because that mode already surfaces thread roots by design.
- **Preamble deleted, not gated.** The `fmt.Fprintf(os.Stderr, "Showing %d comments.\n", ...)` line was removed unconditionally. It was purely informational and polluted piped/scripted output.
- **No `--all` escape hatch.** The PRD and ACs don't call for one. If a future caller needs all comments including replies, they can use `--thread` on a specific comment anchor.

### Files changed

- `server/cmd/multica/cmd_issue.go` — removed preamble; added roots-only filter guard before cursor/output block.
- `server/cmd/multica/cmd_issue_test.go` — added `captureStdout` helper; `issueCommentListServer` helper; three new tests: `TestRunIssueCommentList_NoPreamble`, `TestRunIssueCommentList_DefaultFiltersRootsOnly`, `TestRunIssueCommentList_ThreadModeIncludesReplies`.

### Verification

- `go test ./cmd/multica/` — 245 passed (includes 3 new tests).
- `pnpm test` — 680 TS tests passed.
- `pnpm typecheck` — 0 errors.
