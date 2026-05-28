# Issue 11: Skill overrides — `.claude/skills/to-prd.md` and `.claude/skills/to-issues.md` call MCP

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/feature-pipeline/PRD.md`

## What to build

Project-scoped overrides of the global `/to-prd` and `/to-issues` skills so they persist into Multica via MCP instead of writing markdown into `.scratch/`. `/grill-me` is intentionally not overridden — it produces conversation context that `/to-prd` consumes.

**Files**:
- `.claude/skills/to-prd.md` — local override of the global skill.
- `.claude/skills/to-issues.md` — local override of the global skill.

**`/to-prd` behavior** (overridden):
- Read the conversation context, synthesize a PRD body (same content rules as the global skill).
- Call `mcp.create_feature` with: `title` (derived from the PRD topic), `description` (the full PRD body in markdown), `priority` (default `medium`), `target_branch` (set ONLY if the user has expressed a clear preference for one PR per feature; otherwise leave NULL).
- Return the new feature's identifier (e.g. `MUL-F-12`) to the user with a short summary: "Created feature MUL-F-12 (status: planned). Approve when ready: ask me or click in dashboard."
- Do NOT call `approve_feature` automatically — approval is a deliberate user act.

**`/to-issues` behavior** (overridden):
- Read the parent feature reference from the user (the identifier from `/to-prd` or a manually specified one). Fetch it via `mcp.get_feature` to ground decomposition in the actual stored description.
- Decompose into vertical tracer-bullet slices (same rule as the global skill).
- Quiz the user with the proposed breakdown (slices, models, dependencies) — same interview pattern as the global skill.
- Once approved, call `mcp.create_issue` once per slice with `feature_id` set. Capture each returned issue identifier.
- For each sequential edge in the approved breakdown, call `mcp.link_issue_dependency(issue_id, depends_on_issue_id, type='blocks')`.
- Report back: "Created N issues under feature MUL-F-12. Dependencies linked. The motor will start when you approve the feature."

**Error handling**: if any MCP call fails partway through, the skill MUST surface the partial state ("created issues 1, 2, 3 then failed creating 4 because: …") so the user knows what's in Multica and what isn't. No silent rollback — that requires transactional MCP tools which the v1 server doesn't provide.

## Acceptance criteria

- [ ] `.claude/skills/to-prd.md` exists and overrides the global skill (Claude Code prefers project-scoped skill files over global ones).
- [ ] `.claude/skills/to-issues.md` exists and overrides the global skill.
- [ ] Running `/to-prd` in a Claude Code session inside this repo creates a `feature` row via MCP (not a `.scratch/` markdown file).
- [ ] Running `/to-issues <feature-id>` creates issues under the named feature via MCP and links dependencies.
- [ ] The overrides preserve the interview/approval cycle of the global skills (no silent auto-publish).
- [ ] Partial failure surfaces what was and wasn't created.
- [ ] The original global skill behavior is unchanged outside this repo (other projects still write to `.scratch/`).

## Blocked by

- `.scratch/feature-pipeline/issues/09-mcp-feature-write-tools.md`
- `.scratch/feature-pipeline/issues/10-mcp-issue-write-tools.md`

## Comments

### Key decisions made

1. **Flat file format** — Used `.claude/skills/to-prd.md` and `.claude/skills/to-issues.md` (flat files in `.claude/skills/`) as specified in the issue, rather than the global convention of `<name>/SKILL.md` directories. Flat files are simpler and the issue explicitly names the target paths.

2. **`to-prd` does not auto-approve** — The skill explicitly tells the agent not to call `approve_feature` after creating the feature, preserving the deliberate approval ritual ("the motor doesn't start until you say so").

3. **`to-issues` captures UUIDs for dependency linking** — The skill instructs the agent to capture the UUID returned by each `create_issue` call (not just the human-readable identifier) because `link_issue_dependency` requires UUIDs. The agent is told to pass `get_feature`'s `id` field as `feature_id`.

4. **Dependency order enforced** — The skill explicitly says "publish in dependency order (blockers first)" so real UUIDs are available when `link_issue_dependency` is called — same as the global skill's publishing order requirement.

5. **Partial failure is surfaced, not swallowed** — The skill instructs the agent to stop and report partial state immediately on any MCP failure, including what was created (with UUIDs) so the user can resume manually. No automatic rollback since v1 MCP has no transactional tools.

6. **PRD template preserved** — Both skills carry the same PRD and issue description templates as the global skills, ensuring the content quality expectations are identical — only the persistence destination changes (MCP instead of `.scratch/`).

### Files changed

- `.claude/skills/to-prd.md` — new file: project-scoped override of the global `/to-prd` skill; calls `create_feature` via MCP
- `.claude/skills/to-issues.md` — new file: project-scoped override of the global `/to-issues` skill; calls `create_issue` + `link_issue_dependency` via MCP

### Blockers or notes for next iteration

None. Skills are markdown files consumed at runtime; no Go/TS tests apply. `pnpm typecheck` and `pnpm test` both pass (unchanged). End-to-end verification requires a running backend with the MCP server configured (`claude mcp add multica -- multica mcp`) — not automatable in CI.
