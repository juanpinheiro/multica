# Fork multica and subtract, rather than greenfield

We build on the existing multica fork and delete what we don't need, rather than
starting a new project and copying multica in as reference.

The execution core we want to keep — the daemon, the Postgres-backed task queue and
claim gate, the MCP server, worktree handling, crash recovery — is woven through the Go
server and schema by `workspace_id`. Copying it into a clean repo would mean
re-deriving that entanglement by hand and reintroducing integration bugs. The
multi-tenant model is already neutered to harmless "folders" (a singleton implicit
user), so carrying it costs almost nothing, while the parts we discard (squad,
login/user-identity chrome, the `member` assignee type) are clean, reviewable
subtractions.

The license (a modified Apache 2.0) permits this for internal/personal use; only
offering it as a hosted service or embedding it in a product sold to third parties
would require a commercial license. The Apache attribution (NOTICE/headers) must be
preserved even though the product may later be renamed.

## Considered Options

- **Greenfield + copy multica as reference** — rejected. Re-deriving the entangled
  daemon/server/schema/MCP is strictly more work and more risk than deleting the
  unwanted ~15%. The "smaller scope" intuition was wrong: we cut squad + multi-tenant
  chrome but *add* orchestrator, validators, contracts, and self-evolving memory — the
  core grows, it doesn't shrink.
