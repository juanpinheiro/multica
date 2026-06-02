# Issue 07: Claude Code plugin scaffold + marketplace

**Status:** `ready-for-agent`
**Model:** `sonnet`

## Parent

`.scratch/standalone-install/PRD.md`

## What to build

Make the repo installable as a Claude Code plugin via the native marketplace mechanism: `claude plugin marketplace add multica-ai/multica` → `claude plugin install multica`. The plugin bundles the control-plane planning skills (multica-aware `/to-prd`, `/to-issues`, `/triage`), the `/setup-multica` command (Issue 05), and **one globally-declared `multica mcp` server** (which resolves the Workspace from the cwd manifest per Issue 04).

Build the `.claude-plugin/marketplace.json` + `plugin.json` declaring the bundled skills, the command, and the MCP server, so installing the plugin makes the skills and the multica MCP appear in Claude Code without manual config edits.

## Acceptance criteria

- [ ] Adding the marketplace and installing the plugin registers the planning skills and `/setup-multica` in Claude Code.
- [ ] The plugin declares a single global `multica mcp` server; after install it appears as an MCP server without manual `claude mcp add`.
- [ ] Launching Claude Code in a configured Umbrella targets that Umbrella's Workspace via the cwd-resolving MCP (Issue 04) — verifying the end-to-end "opens in the right Workspace" path.
- [ ] The plugin manifest is valid against the marketplace/plugin schema the project already uses (consistent with the user's installed marketplaces).
- [ ] A short doc covers `claude plugin marketplace add` → `claude plugin install multica` and the per-project `/setup-multica` step.

## Blocked by

- Issue 04 (MCP cwd→Workspace resolver)
- Issue 05 (`/setup-multica` command)
