# Issue 21: Agent MCP-config tab (optional)

**Status:** `done`
**Model:** `sonnet`

## Parent

PRD 1 — Upstream Sync (`.scratch/upstream-sync/PRD.md`).

## What to build

Optional, lowest-priority. Add a tab on the agent detail page for configuring the MCP servers an agent can consume (including ACP runtimes), ported from upstream. This is distinct from the fork's own `multica mcp` server (which is server-side, from the feature-pipeline) — this tab is about an agent *consuming* MCP servers. Pull only if per-agent MCP config is wanted now.

## Acceptance criteria

- [x] The agent detail page has an MCP-config tab for declaring MCP servers the agent consumes.
- [x] The tab is shown for ACP runtimes as well (no runtime-specific gate — renders for all agents regardless of runtime).
- [x] Config persists and is surfaced to the daemon/agent run as today upstream (server-side `mcp_config` jsonb and `--mcp-config` injection already existed; the tab reads/writes through the existing `PUT /api/agents/{id}` endpoint).
- [x] View test covers the tab rendering and save flow (8 tests).

## Blocked by

None — can start immediately (optional; lowest priority).

## Comments

### Key decisions

- **No new backend endpoints.** The `mcp_config` JSONB column and the redaction logic (`mcp_config_redacted`) already existed server-side. The tab reads `agent.mcp_config` / `agent.mcp_config_redacted` from the existing agent response and writes through `PUT /api/agents/{id}` via the `onSave` callback (same pattern as `CustomArgsTab`).

- **Opaque type rather than strict interface.** `mcp_config` is typed as `Record<string, unknown> | null` on both `Agent` and `UpdateAgentRequest`. The server stores and forwards it verbatim; imposing a specific TS shape would add a false contract with no enforcement benefit.

- **Client-side validation with a type-guard, not zod.** `parseConfig` checks the string is valid JSON, is a non-array object, and has a top-level `mcpServers` object key. No zod dependency added to `@multica/views`; plain `JSON.parse` + guard is sufficient and avoids a phantom-dep violation.

- **Redacted-notice path.** When `mcp_config_redacted === true` the tab renders a prose notice instead of an editor, consistent with the server's security contract (agent-actor tokens never see the config).

- **Tab shown for all runtimes.** There is no runtime-specific guard — MCP config is an agent property, not a runtime property. The Claude backend is the primary consumer (`--mcp-config`), but the field is forwarded by the daemon layer and other backends that declare `mcpServers` support.

### Files changed

- `packages/core/types/agent.ts` — `mcp_config?: Record<string, unknown> | null` and `mcp_config_redacted?: boolean` added to `Agent`; `mcp_config?: Record<string, unknown> | null` added to `UpdateAgentRequest`.
- `packages/views/locales/en/agents.json` — `tabs.mcp` label and `tab_body.mcp` section (intro, label, placeholder, redacted_notice, invalid_json, invalid_schema, empty_state, saved_toast, save_failed_toast).
- `packages/views/agents/components/tabs/mcp-tab.tsx` (new) — `McpTab` component with JSON textarea editor, validation, redacted notice, and dirty-state tracking.
- `packages/views/agents/components/tabs/mcp-tab.test.tsx` (new) — 8 tests: existing config renders, redacted notice, empty state, dirty change, invalid JSON error, schema error, save with parsed config, save with null on clear.
- `packages/views/agents/components/agent-overview-pane.tsx` — `"mcp"` added to `DetailTab` union, `TAB_LABEL_KEY`, `detailTabs` array (with `Server` icon from lucide-react), and tab render block.

### Verification

- `pnpm --filter @multica/views exec vitest run agents/components/tabs/mcp-tab.test.tsx` — 8/8 passed.
- `pnpm typecheck` — 4 packages, 0 errors.
- `pnpm test` — 698 tests, 84 files, all green.
