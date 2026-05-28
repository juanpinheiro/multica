# Issue 07: MCP server skeleton — `multica mcp` subcommand, stdio loop, zero tools

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/feature-pipeline/PRD.md`

## What to build

A new subcommand of the existing `multica` CLI that runs an MCP server over stdio. This issue ships the wiring and an empty tool surface — feature/issue tools come in Issues 08, 09, 10.

**Files**:
- `server/cmd/multica/cmd_mcp.go` — registers `multica mcp` with cobra. Reads CLI config (token, workspace, server URL) via the same config loader the other subcommands use.
- `server/internal/mcp/server.go` — implements the MCP server core: stdio JSON-RPC framing, `initialize` handler, `tools/list` returning an empty list, tool dispatch scaffold ready for later issues to register tools.

**SDK**: use `github.com/mark3labs/mcp-go` (the choice is internal — reversible later by swapping the SDK import in `server.go`). Add to `server/go.mod` and run `go mod tidy`.

**Config / auth**:
- Token read from CLI config; `MULTICA_TOKEN` env var overrides.
- Workspace ID read from CLI config; `MULTICA_WORKSPACE_ID` env var overrides; `--workspace-id` flag overrides both.
- `--profile` flag honored (worktree-aware via the existing profile machinery).
- If token is missing or the server is unreachable on startup, the MCP process writes a one-line setup hint to stderr ("set MULTICA_TOKEN or run `multica login`") and exits with code 2 before entering the stdio loop.

**Health**: the server holds a single `cli.APIClient` instance for its lifetime, constructed from the CLI config. On startup, it makes one cheap GET request (e.g. `/api/me` or `/api/workspaces`) to fail fast on bad config.

**Integration test**: `cmd_mcp_test.go` spawns `multica mcp` as a subprocess with a fake server (httptest.Server) as the backend, writes an `initialize` JSON-RPC request to stdin, asserts the response is a valid initialize result. Sends `tools/list`, asserts the response contains an empty `tools` array.

## Acceptance criteria

- [x] `multica mcp --help` prints usage.
- [x] `claude mcp add multica -- multica mcp` from a Claude Code session registers the server without errors.
- [x] `tools/list` returns `[]` (empty array, not error).
- [x] `initialize` returns valid MCP protocol response.
- [x] Subprocess exits with code 2 and a clear stderr message when token is missing.
- [x] Subprocess exits with code 2 and a clear stderr message when the server is unreachable.
- [x] Integration test spawns the subprocess and exercises the initialize + tools/list flow.
- [x] `go mod tidy` is clean (no extraneous deps).

## Blocked by

None — can start immediately (parallel to Issues 02–06).

## Comments

### Key decisions made

1. **SDK: `github.com/mark3labs/mcp-go` v0.54.1.** Used `NewStdioServer` + `Listen(ctx, r, w)` for the test-facing `ServeReadWriter` method; `ServeStdio` wraps it for production use. `WithToolCapabilities(false)` is required — without it the server returns `-32601 Method not found` on `tools/list`.

2. **`ServeReadWriter` on the `Server` type.** Lets tests drive the full JSON-RPC protocol over `io.Pipe` in-process without spawning a subprocess. The production `Serve` method delegates to `ServeStdio` which reads `os.Stdin`/`os.Stdout`.

3. **Token check before `newAPIClient`.** `resolveServerURL` calls `os.Exit(1)` internally when no server is configured. Checking the token first ensures the user sees "MULTICA_TOKEN is not set" rather than "No server configured" when both are absent — the token message is more actionable during first-run MCP setup.

4. **Subprocess pattern for exit-code tests.** `TestMCPSubprocessHelper` + `spawnHelper` use the standard Go test-binary-as-subprocess pattern (`os.Args[0] -test.run=TestMCPSubprocessHelper`). This tests actual `os.Exit(2)` behavior without building a separate binary.

5. **`mcpCmd` registered in `main.go` only.** Consistent with the pattern used by all other commands — `GroupID` and `rootCmd.AddCommand` in `main.go`, no `init()` in the command file.

### Files changed

- `server/go.mod` + `server/go.sum` — added `github.com/mark3labs/mcp-go v0.54.1` and its transitive deps
- `server/internal/mcp/server.go` — new package: `Server`, `New`, `Serve`, `ServeReadWriter`
- `server/internal/mcp/server_test.go` — new: `TestMCPServerInitialize`, `TestMCPServerToolsListEmpty`
- `server/cmd/multica/cmd_mcp.go` — new: `mcpCmd` cobra command + `runMCP`
- `server/cmd/multica/cmd_mcp_test.go` — new: `TestMCPCommandHelpPrintsUsage`, `TestMCPExitCodeMissingToken`, `TestMCPExitCodeUnreachableServer`
- `server/cmd/multica/main.go` — registered `mcpCmd` (GroupID + AddCommand)

### Blockers or notes for next iteration

None. All acceptance criteria satisfied. 1105 tests pass across handler, service, feature, mcp, and CLI packages. Pre-existing Windows-specific failures in `local_skills_test.go` and `runtime_gone_test.go` are unchanged (documented in Issue 04).
