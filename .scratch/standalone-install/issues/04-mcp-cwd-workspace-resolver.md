# Issue 04: MCP cwd→Workspace resolver

**Status:** `ready-for-agent`
**Model:** `sonnet`

## Parent

`.scratch/standalone-install/PRD.md`

## What to build

Teach the `multica mcp` server to resolve the active **Workspace from the current working directory's `.multica/workspace.toml`**, so a single globally-registered MCP server follows whatever folder Claude Code is launched in. This is the one net-new control-plane path the standalone install needs (ADR-0008, option A).

Today `multica mcp` resolves the Workspace from the CLI profile default. Add a resolver that, on MCP startup, walks up from the working directory using the existing `manifest.Find` to locate `.multica/workspace.toml`, parses its `workspace` slug, and resolves that slug to a Workspace on the running server. Resolution order: explicit flag/env override → cwd manifest slug → clear error if neither yields a Workspace.

The resolver must be a deep module that is pure over its inputs (start directory, a filesystem reader, and a slug→Workspace lookup) so it can be tested without a live server, then wired into the `multica mcp` boot path.

## Acceptance criteria

- [ ] With a `.multica/workspace.toml` at or above the cwd whose slug exists on the server, `multica mcp` targets that Workspace automatically.
- [ ] An explicit flag/env Workspace override takes precedence over the cwd manifest.
- [ ] No manifest found at or above the cwd → a clear error (or documented explicit-default fallback), not a silent wrong-Workspace.
- [ ] A manifest whose slug is absent on the server → a precise error naming the slug and pointing at `/setup-multica`.
- [ ] The core resolution logic is unit-tested table-driven over injected FS + slug→Workspace lookup, with no live server. Prior art: `workspace/manifest/manifest_test.go`, `workspace/reconcile/reconcile_test.go`.
- [ ] `make check` passes.

## Blocked by

- None - can start immediately
