# Issue 15: Adapt /to-prd and /to-issues to emit the new entities

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/initiative-runner/PRD.md`

## What to build

Update the planning skills so the control plane produces the new model through the MCP: `/to-prd` emits an
Initiative with a Definition of Done and a Mode (HITL/AFK); `/to-issues` carves it into Milestones and
Issues, each Issue carrying a `Model` preference. The skills write via the MCP control-plane tools rather
than freehand markdown.

## Acceptance criteria

- [ ] `/to-prd` produces an Initiative + DoD + Mode via the MCP
- [ ] `/to-issues` produces ordered Milestones and Issues with `Model` set per Issue
- [ ] A planned Initiative is creatable and `ready`-able entirely through the skills
- [ ] `pnpm test` passes

## Blocked by

- `14-mcp-control-plane`

## Comments

### Key decisions

1. **DoD as text in /to-prd, assertions as DB records in /to-issues.** DoD assertions require a `milestone_id`, so they cannot be created until milestones exist. `/to-prd` adds a "Definition of Done" section to the PRD template (text in the description field) and captures Mode via `create_feature`; `/to-issues` creates the per-Milestone DB records via `create_milestone` + `create_dod_assertion`.

2. **Mode (hitl/afk) clarified in /to-prd step 4.** The skill explains the per-Initiative choice with a note that HITL = split into several Initiatives vs AFK = one big Initiative. The `mode` parameter is passed to `create_feature`.

3. **Fixed stale `status: planned` reference** in `/to-prd` — corrected to `status: draft` following the issue-06 status reshape.

4. **Publish order in /to-issues: Milestones → DoD assertions → Issues.** This ordering is required because DoD assertions need milestone UUIDs and issues need milestone UUIDs. Step 7 is split into 7a/7b/7c to make the ordering explicit.

5. **`model` and `milestone_id` added to every `create_issue` call** in /to-issues. The quiz step now surfaces model recommendations per issue and Milestone grouping for user approval.

6. **Partial-failure resume instructions updated** for /to-issues to mention milestone_id in the resume hint.

### Files changed

- `.claude/skills/to-prd.md` — Mode clarification step, DoD section in PRD template, `mode` param in `create_feature`, fixed status label
- `.claude/skills/to-issues.md` — Milestone grouping (step 4), DoD assertion drafting (step 5), updated quiz (step 6), publish in Milestones→DoD→Issues order (step 7) with `milestone_id`/`model` on issues, updated report-back

### Blockers / notes

None. `pnpm test` (677 tests) and `pnpm typecheck` both pass. No code changes — skill files are markdown instructions only.
