# ISSUES

The feature directory's markdown files are provided at start of context, each
prefixed with a `=== FILE: <path> ===` marker. There are two kinds:

- `<feature>/PRD.md` — the parent product doc. **Not an issue.** Read it for
  context, but ignore any `Status:` line on it. Never mark it `done`.
- `<feature>/issues/<NN>-<slug>.md` — the actual implementation issues. Each
  starts with a `Status:` line. The canonical statuses are defined in
  `docs/agents/triage-labels.md`.

Only consider files under `issues/` with `Status: ready-for-agent`. Ignore
everything else (`ready-for-human`, `done`, `needs-triage`, `needs-info`,
`wontfix`).

If there are no `issues/*.md` files with `Status: ready-for-agent` left,
output <promise>NO MORE TASKS</promise> and stop.

# TASK SELECTION

The ralph script pre-selects a single issue and injects its path with a
`=== ASSIGNED ISSUE: <path> ===` marker in the prompt. Work on **only** that
issue — do not pick a different one even if you think another is higher
priority. The script enforces ordering and model choice, not you.

If no `=== ASSIGNED ISSUE: ===` marker is present, fall back to picking the
lowest-numbered `ready-for-agent` issue under `<feature>/issues/`.

# EXPLORATION

Explore the repo.

# IMPLEMENTATION

Use /tdd to complete the task. Apply clean code principles while writing
(small functions, intention-revealing names, no comments unless WHY is
non-obvious, SRP). The codebase's existing CLAUDE.md and any sibling
files in the same package set the local style — match them.

# FEEDBACK LOOPS

Before updating the issue status:

- Run /clean-code over the diff and apply its Implementation Checklist
  (functions <20 lines, single responsibility, no redundant comments,
  no dead code, no backwards-compat hacks). Iterate until the checklist
  passes — do not declare done with checklist failures.
- Run:

  - `npm run test`
  - `npm run typecheck`

# STATUS UPDATE

This is the contract that drives the loop. On the next iteration the script
re-reads each issue's `Status:` line to decide what's left.

If the task is complete:

- Change the issue's `Status:` line to `Status: done`
- Append a `## Comments` section at the bottom of the issue summarising:
  1. Key decisions made
  2. Files changed
  3. Blockers or notes for the next iteration

If the task is NOT complete:

- Leave the `Status:` line as `ready-for-agent`
- Append a `## Comments` note describing what was done so far and what's
  blocking completion, so the next iteration can pick up where you left off

# FINAL RULES

ONLY WORK ON A SINGLE TASK.
