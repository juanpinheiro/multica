# Issue 05: Manifest machinery — resolver + scanner + reconciler (pure modules)

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/multi-repo-features/PRD.md`

## What to build

Three pure Go modules under `internal/workspace/` that implement the `.multica` manifest layer. All logic runs over injected interfaces (a filesystem `FS` and a `GitRunner`) so tests use in-memory fakes — no OS or git calls in the core functions. No wiring into CLI/MCP here (that is Issue 06); this slice delivers the tested building blocks.

**Manifest resolver (`internal/workspace/manifest`):**
- `Find(startDir string, fs FS) (manifestPath string, found bool)` — walk up parent directories from `startDir` until `.multica/workspace.toml` is found or the filesystem root is reached.
- `Parse(data []byte) (Manifest, error)` — decode the TOML.
- `Manifest{ Workspace string; Repos []RepoEntry }`, `RepoEntry{ Name, Path, Remote string }`. `Path` is relative to the manifest's directory (the umbrella root).

Manifest shape:
```toml
workspace = "meu-produto"
[[repo]]
name   = "backend"
path   = "./backend"
remote = "github.com/voce/backend"
```

**Repo scanner (`internal/workspace/scan`):**
- `Scan(root string, maxDepth int, fs FS, git GitRunner) []Candidate` where `Candidate{ Name, Path, Remote string }`.
- Walks children up to `maxDepth` (default 2). A directory containing `.git` is a candidate; remote read via `git remote get-url origin` through the injected runner.
- Skips `node_modules`, `vendor`, `dist`, `build`, `.git` internals, and does **not** descend into a discovered repo to find nested repos.

**Reconciler (`internal/workspace/reconcile`):**
- `Reconcile(m Manifest, srv WorkspaceState) Plan` where `Plan{ CreateWorkspace bool; ReposToCreate []RepoEntry; ReposMissingOnDisk []string; ReposOrphanOnDisk []Candidate }`.
- `CreateWorkspace` true when the manifest's workspace slug is absent on the server. `ReposToCreate` = manifest repos absent on server. `ReposMissingOnDisk` = manifest repos whose path does not exist. `ReposOrphanOnDisk` = scanned repos not in the manifest.

## Acceptance criteria

- [ ] Resolver table-driven test: no manifest → not found; manifest in cwd; manifest several levels up; root boundary; malformed TOML → error; relative repo paths resolved against manifest dir; single-repo manifest inside a repo root.
- [ ] Scanner table-driven test: flat umbrella with 3 child repos → 3 candidates; depth bound respected; `node_modules`/`vendor` skipped; no descent into a discovered repo; remote read via fake git runner; child dir with no `.git` ignored.
- [ ] Reconciler table-driven test: workspace absent → `CreateWorkspace=true`; manifest repo absent on server → `ReposToCreate`; manifest path missing → `ReposMissingOnDisk`; orphan repo on disk → `ReposOrphanOnDisk`; fully in-sync manifest → empty plan.
- [ ] No OS/git calls in core functions (verified by the tests running purely on fakes).
- [ ] `make check` passes.

## Blocked by

- Issue 01 (the reconciler's `WorkspaceState` mirrors the server `repo` shape).

## Comments

### Iteration 1 — implemented (Sonnet)

**Key decisions**

- **`FS` interface per package, single responsibility.** The manifest package's `FS` has one method (`ReadFile`); the scan package's `FS` has two (`ReadDir`, `DirExists`). Each interface is minimal — no shared abstraction forced across packages.
- **Walk-up uses `filepath.Dir` loop.** `Find` walks from `startDir` toward the filesystem root by repeatedly calling `filepath.Dir`, stopping when the path stops changing (root boundary). Cross-platform since `filepath` handles both `/` and `\`.
- **`ManifestDir` helper** strips `.multica/workspace.toml` suffix from the manifest path to return the umbrella root. Used by callers that need to resolve relative `repo.Path` values.
- **Reconciler is pure with pre-computed disk presence.** `WorkspaceState` carries `RepoDiskPresence map[string]bool` rather than accepting an FS directly. The caller (Issue 06's wiring layer) resolves disk presence before calling `Reconcile`, keeping the reconciler a pure data-transformation function with zero I/O.
- **Scanner does not descend into discovered repos.** Once a `.git` child is found, it is added as a candidate and the walk does not recurse into it — prevents nested repo discovery.
- **TOML via `pelletier/go-toml/v2`** (already in go.mod from prior work).

**Files created**

- `server/internal/workspace/manifest/manifest.go` — `FS`, `RepoEntry`, `Manifest`, `Find`, `Parse`, `ManifestDir`.
- `server/internal/workspace/manifest/manifest_test.go` — 5 `Find` cases, 4 `Parse` cases, `ManifestDir`, path resolution; all on `memFS` (map-based).
- `server/internal/workspace/scan/scan.go` — `Entry`, `FS`, `GitRunner`, `Candidate`, `skipDirs`, `Scan`, `walk`.
- `server/internal/workspace/scan/scan_test.go` — 7 table-driven cases covering flat umbrella, depth bound, skip-listed dirs, no-descent, no-.git ignored, remote read, no-origin.
- `server/internal/workspace/reconcile/reconcile.go` — `WorkspaceState`, `Plan`, `Reconcile`.
- `server/internal/workspace/reconcile/reconcile_test.go` — 10 table-driven cases covering all Plan fields.

**Verification**

- `go build ./internal/workspace/...`: clean.
- `go test ./internal/workspace/... -v`: 32 tests across 3 packages, all pass.
- `go vet ./internal/workspace/...`: clean.
- `make check`: PASS (typecheck, lint, Go tests, Vitest all green).
