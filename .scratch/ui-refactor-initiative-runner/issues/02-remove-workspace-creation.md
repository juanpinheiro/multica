# Issue 02: Delete workspace-creation flow + NoWorkspacePage empty state

**Status:** `done`
**Model:** `opus`

## Parent

`.scratch/ui-refactor-initiative-runner/PRD.md`

## What to build

Removes the workspace-creation flow entirely and replaces the "no workspace" root state with a `NoWorkspacePage` empty state that points the developer at `/setup-multica`.

End-to-end behavior after this slice:

- A signed-in user with zero workspaces lands on `/` and sees `NoWorkspacePage`: a short copy explaining no workspace was discovered, the literal `/setup-multica` command in a code block, and a link to the standalone-install docs. The page does not offer a name/slug form.
- A signed-in user with at least one workspace and a `last_workspace_slug` cookie still gets redirected to that workspace's Live page (root cookie behaviour preserved).
- `/workspaces/new` returns 404.
- The codebase no longer carries `CreateWorkspaceForm`, `NewWorkspacePage`, the `useCreateWorkspace` mutation, `paths.newWorkspace()`, or the `GLOBAL_PREFIXES` entry for `workspaces`.
- `resolvePostAuthDestination([])` returns `paths.root()` (the empty state); its test is updated.
- `reserved_slugs.json` is regenerated — `workspaces` is no longer reserved if `/workspaces/new` was its only use; if other paths still need it, leave it. The CI generator check passes either way.
- Settings page does not offer workspace creation (verify and remove any residual create paths if found; edit/delete remain).

## Acceptance criteria

- [x] `/workspaces/new` returns 404; the route file no longer exists.
- [x] `CreateWorkspaceForm`, `NewWorkspacePage`, `useCreateWorkspace`, and `paths.newWorkspace` are deleted from the codebase. `rg "useCreateWorkspace|CreateWorkspaceForm|newWorkspace\\(" packages apps` returns nothing.
- [x] Root page `/` renders `NoWorkspacePage` empty state when no `last_workspace_slug` cookie is present and the user has no workspaces.
- [x] Root page `/` continues redirecting to `/{slug}/live` when the cookie points at an existing workspace.
- [x] `NoWorkspacePage` includes the `/setup-multica` command in a code block and a link to standalone-install docs.
- [x] `resolvePostAuthDestination([])` returns `paths.root()`; its test asserts the new expectation.
- [x] `pnpm generate:reserved-slugs` produces no drift.
- [x] `make check` passes.

## Blocked by

- `.scratch/ui-refactor-initiative-runner/issues/01-paths-and-chrome.md` (root redirect target depends on the new `paths.root()` shape from slice 01)

## Comments

### Key decisions

- **`workspaces` slug freed.** With `/workspaces/new` gone there are no remaining `/workspaces/*` routes, so `workspaces` is removed from `reserved_slugs.json`'s pre-workspace-routes group — users can now choose it as a workspace slug. The TS mirror was regenerated; the CI drift check passes.
- **`GLOBAL_PREFIXES` and `isGlobalPath` deleted entirely** (not just emptied). They were a check against `/workspaces/...` shadowing user slugs, but with no global prefixes left the function would always return `false`. The single consumer, `link-handler.ts`, drops the dead check; the segment allowlist (`WORKSPACE_ROUTE_SEGMENTS`) is the only gate that still matters for path rewriting.
- **NoWorkspacePage is self-healing.** When mounted with workspaces in the TQ cache (e.g. the user cleared the cookie but has projects), it `replace`s to the first workspace's `/live` instead of showing the empty state. Keeps `/` recoverable without a manual click.
- **API client method `api.createWorkspace` kept.** The Go backend's `POST /api/workspaces` is still needed by `/setup-multica` running over MCP. Only the React-side mutation/form/page were dead. The existing `client.test.ts` test that uses `createWorkspace` as a 409 fixture remains valid.
- **CLI `multica login` updated to point at `/setup-multica`** instead of opening the now-deleted `/workspaces/new` URL. `tryResolveAppURL` became unused once the browser-open path was removed, so it was deleted too.
- **Proxy legacy-route fallback** (`/{legacy}/...` when there's no `last_workspace_slug` cookie) now redirects to `/` instead of `/workspaces/new`. The root page then renders `NoWorkspacePage`.
- **`paths.workspace(slug).live()`** is used for the cookie-redirect target rather than `.issues()` — slice 01 made `.live()` the canonical landing page; preserving cookie behaviour means honouring the new shape.
- **`resolvePostAuthDestination([])`** changed return only for the empty case. The first-workspace branch still returns `/<slug>/issues` — slice 04 owns the `/issues` → per-initiative-board rename.

### Files changed

- `apps/web/app/page.tsx` — render `<NoWorkspacePage />` when no cookie; cookie path uses `.live()`.
- `apps/web/app/workspaces/new/page.tsx` — deleted (route folder removed).
- `apps/web/proxy.ts` — legacy-route fallback target updated from `/workspaces/new` to `/`.
- `packages/views/workspace/no-workspace-page.tsx` — new component (empty state + self-healing redirect).
- `packages/views/workspace/no-workspace-page.test.tsx` — covers copy, docs link, and self-healing redirect.
- `packages/views/workspace/new-workspace-page.tsx` — deleted.
- `packages/views/workspace/create-workspace-form.tsx` + `create-workspace-form.test.tsx` — deleted.
- `packages/views/workspace/slug.ts` + `slug.test.ts` — deleted (only used by the form).
- `packages/views/locales/en/workspace.json` — drops `create_form` and `new_page` keys; adds `no_workspace` keys.
- `packages/views/package.json` — exports `./workspace/no-workspace-page` (replacing the deleted `./workspace/new-workspace-page`).
- `packages/views/editor/utils/link-handler.ts` — drops `isGlobalPath` check.
- `packages/core/paths/paths.ts` — removes `newWorkspace`, `GLOBAL_PREFIXES`, `isGlobalPath`.
- `packages/core/paths/paths.test.ts` — covers the trimmed surface.
- `packages/core/paths/consistency.test.ts` — drops the global-path-vs-reserved-slug invariant (no global prefixes left to check).
- `packages/core/paths/resolve.ts` — empty workspaces now returns `paths.root()`.
- `packages/core/paths/resolve.test.ts` — asserts the new return value.
- `packages/core/paths/index.ts` — stops re-exporting `isGlobalPath`.
- `packages/core/workspace/mutations.ts` — `useCreateWorkspace` block deleted; `Workspace` import dropped.
- `server/internal/handler/reserved_slugs.json` — drops `workspaces` from the pre-workspace-routes group.
- `packages/core/paths/reserved-slugs.ts` — regenerated.
- `server/cmd/multica/cmd_login.go` — prompts the user to run `/setup-multica` in Claude Code; the browser-open path and `tryResolveAppURL` helper are removed.

### Notes for the next iteration

- `apps/web/proxy.ts` still carries the `LEGACY_ROUTE_SEGMENTS` set (`issues`, `features`, etc.). Slice 04 (URL rename `/features` → `/initiatives`) should revisit which legacy segments still need redirection.
- `client.ts:createWorkspace` and `client.test.ts` keep an API client method and a 409-handling test for it. The Go endpoint exists for `/setup-multica` via MCP — leave it as the network-layer contract for that flow.
- `WORKSPACE_SLUG_REGEX` lived in `packages/views/workspace/slug.ts` (now deleted). If a future surface needs workspace-slug validation client-side (e.g. an MCP scaffold preview), recreate it next to that caller — the previous form-only home of the regex was the only consumer.
