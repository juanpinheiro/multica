# PRD: Multi-Repo Features — workspace-as-context, first-class repos, and the `.multica` manifest

**Status:** `ready-for-agent`
**Owner:** Juan Pinheiro
**Created:** 2026-05-29
**Depends on:** `.scratch/feature-pipeline/PRD.md` (the feature pipeline must land first — this PRD amends several of its decisions)
**Amends:** `.scratch/feature-pipeline/PRD.md` — specifically: `feature.target_branch` (single branch → derived per `(feature, repo)`), the branch resolver signature (`Resolve(issue, feature)` → `Resolve(issue, feature, repo)`), the branch gate in the claim handler (per-feature → per-`(repo, branch)`), and PR consolidation (one PR per feature → one PR per repo touched by the feature).

## Problem Statement

The feature pipeline (the prior PRD) gives the user a clean PRD-to-PR loop: `/to-prd` creates a `feature`, `/to-issues` decomposes it into issues with dependency edges, the daemon picks them up in order, and the work converges on a single branch and a single PR. That model assumes one thing that does not hold in the user's real workflow: **a feature lives in exactly one repository.**

The user's actual work is cross-repo. A single feature — say "auth-v2" — has a backend slice, a frontend slice, and a QA slice, each landing in a *different* git repository, each producing its *own* PR. The prior PRD's `feature.target_branch` (a single branch) and its branch resolver (`Resolve(issue, feature)`, no repo dimension) cannot express this. A feature that touches three repos would need three target branches and three PRs, and the scheduler's branch-serialization gate would incorrectly serialize work across unrelated repos.

There is a second, related problem the user surfaced during design: **the "workspace" concept feels redundant and its developer experience is unclear.** Multica's `workspace` was originally a multi-tenant boundary. In the personal fork the tenant boundary already collapsed to a singleton implicit user, so "workspace" lost its original meaning and became an abstract label the user has to invent (`personal` / `work` / `oss`) and carry around. Two questions had no good answer:

1. **What is a workspace *for*, now that there is only one user?** If it were the repo, a cross-repo feature would have to span three workspaces, which is incoherent — a feature is one thing.
2. **How does the user select the active workspace without ceremony?** Passing `--workspace` on every Claude invocation is unacceptable DX. There was no mechanism that resolved the active workspace from where the user already is.

The combined effect: the feature pipeline as specified cannot model the user's real (multi-repo) features, and the workspace concept around it has no clear role or ergonomics. Both must be resolved together, because they are the same modeling question seen from two sides — *what is the unit that groups repos, and how is it discovered?*

## Solution

Separate the two concepts the old `workspace` was conflating, and add the filesystem-anchored ergonomics that make the whole thing zero-ceremony to use.

**1. Re-define `workspace` as a context that groups repositories.** A workspace is no longer a tenant boundary and is not a repo. It is the cabinet that holds a set of related repos plus the agent roster and the cost ledger that operate across them. Examples: a workspace "meu-produto" holds the `backend`, `frontend`, and `qa` repos; a separate workspace "side-projects" holds unrelated repos. This rescues the workspace concept by giving it the one job nothing else can do: be the level *above* repos, the level a cross-repo feature lives in.

**2. Promote `repo` to a first-class entity.** Today a workspace's repos live in a `workspace.repos jsonb` blob. Promote them to a `repo` table with a stable id, a `remote_url` (the resolution key), and a `local_path` (where the daemon checks out). An `issue` gains `repo_id`: the single repo that issue's code lands in. A `feature` no longer carries a single target branch — instead, the branch is *derived per `(feature, repo)`*, so the same feature produces `feature/auth-v2` in the backend repo, `feature/auth-v2` in the frontend repo, and `feature/auth-v2` in the QA repo, each accumulating its own PR.

**3. A feature spans repos within a workspace and aggregates N PRs.** `/to-issues` tags each issue with the repo it targets, drawn from the workspace's known repo list. The daemon checks out the right repo per issue and pushes to that repo's feature branch. The branch-serialization gate keys on `(repo_id, branch)`, so backend and frontend issues run *in parallel* (different repos) while two backend issues sharing the backend feature branch run *serially*. The feature view aggregates the PRs — one per repo touched — instead of expecting a single PR.

**4. The `.multica` manifest makes workspace selection zero-ceremony.** A workspace is anchored to a filesystem **umbrella directory** that contains its repos as children. That directory holds a `.multica/workspace.toml` manifest listing the workspace slug and its repos. Resolving the active workspace becomes a *walk-up* from the current directory until a `.multica` is found — exactly how git finds `.git`, how pnpm finds `pnpm-workspace.yaml`, how `go.work` anchors a Go multi-module workspace. The user never passes a workspace flag; they open Claude wherever they already are and the manifest above them determines the context, offline, with no server round-trip.

```
~/code/meu-produto/
  ├─ .multica/workspace.toml      # workspace = "meu-produto" + repo list
  ├─ backend/    (.git)
  ├─ frontend/   (.git)
  └─ qa/         (.git)
```

**5. Onboarding is a scan-and-suggest, not a command to remember.** When a session starts and no `.multica` is found by walking up, a setup skill performs a shallow scan of the current directory for child git repos and *proposes* a workspace: "Found 3 repos here — backend, frontend, qa. Create workspace 'meu-produto' with these?" One confirmation writes the manifest, creates the workspace, and registers the repos. The manifest is the source of truth for composition; the server is a projection of it, reconciled on session start. Because the manifest pins the workspace slug, re-reads are idempotent, and a fresh machine or a wiped DB can rebuild the entire workspace from the file.

The result: the user works in any subdirectory of their umbrella folder, the correct workspace and its repos are already active with zero input, a single PRD decomposes into issues across the backend/frontend/QA repos, agents run in parallel where the repos differ and serially where they share a branch, and each repo accumulates its own coherent PR under one feature.

## User Stories

### Workspace as context

1. As a solo engineer, I want a workspace to represent a *context that groups several repositories* (e.g. backend + frontend + qa for one product), so that I have a home for cross-repo features instead of being forced to pick a single repo.
2. As a solo engineer, I want to keep more than one workspace (e.g. "meu-produto" and "side-projects"), each with its own set of repos, agent roster, and cost ledger, so that unrelated work stays isolated.
3. As a solo engineer, I do NOT want a workspace to mean a single repository, because my features routinely span multiple repos and a feature is one coherent thing.
4. As a solo engineer, I want the issue key prefix (e.g. `MUL-123`) to remain workspace-level, so that one feature's backend and frontend PRs both reference the same `MUL-123` and the key is traceable across repos.
5. As a solo engineer, I want the agent roster and cost tracking to belong to the workspace (shared across its repos), so that I see one budget for the product rather than a fragmented per-repo view.

### First-class repos

6. As a solo engineer, I want each repository to be a first-class entity in Multica (not a blob field), with a name, a git remote, a local path, and a default branch, so that issues and PRs can reference a specific repo.
7. As a solo engineer, I want an issue to target exactly one repository, so that the daemon knows which repo to check out and where the resulting PR goes.
8. As a solo engineer, I want an issue with no repository (rare — e.g. a coordination/tracking issue) to be allowed and simply skipped by the daemon's checkout logic, so that not every issue is forced to produce code.
9. As a solo engineer, I want to list the repositories of the active workspace from Claude, so that `/to-issues` can assign each slice to a repo from a known menu instead of me typing repo names.

### Cross-repo features

10. As a solo engineer, I want a feature (PRD) to span multiple repositories within its workspace, so that "auth-v2" can have backend, frontend, and QA slices under one spec.
11. As a solo engineer, I want each repository touched by a feature to get its own branch named `feature/<slug>`, so that the work in each repo is isolated and reviewable on its own.
12. As a solo engineer, I want each repository touched by a feature to accumulate its own pull request, so that I review one cohesive PR per repo instead of one giant cross-repo PR (impossible) or fragmented per-issue PRs.
13. As a solo engineer, I want the feature's branch name to be identical across repos (`feature/auth-v2` everywhere), so that the feature is visually traceable and I can reason about it as one change.
14. As a solo engineer, I want issues targeting *different* repos to be dispatched in parallel even if they share the same branch name, so that backend and frontend work proceeds concurrently.
15. As a solo engineer, I want issues targeting the *same* repo on the same feature branch to be serialized, so that two agents never push to the same branch of the same repo at once.
16. As a solo engineer, I want dependency edges (e.g. "frontend issue blocked by backend issue") to be honored across repos, so that the frontend agent only starts once the backend contract exists.
17. As a solo engineer, I want the feature view to show all N PRs (one per repo) with their statuses, so that I see the whole feature's progress in one place.
18. As a solo engineer, I want the feature view to group issues by repo and then by dependency layer (ready / blocked), so that I see at a glance what the daemon will pick up next in each repo.

### Cross-repo agent context

19. As a solo engineer, I want the agent working a backend issue to receive the feature's shared PRD as context, so that every slice is implemented against the same spec.
20. As a solo engineer, I want each issue's brief to summarize its sibling issues and which repos they target, so that the agent understands its slice's place in the cross-repo feature.
21. As a solo engineer, I want the agent to receive relevant cross-repo contracts as text (e.g. the API schema the frontend depends on), so that the frontend slice is built against the real backend interface.
22. As a solo engineer, I want the option (later) to mount read-only checkouts of sibling repos alongside the agent's worktree, so that an agent can `grep` a sibling repo when text context is not enough.

### The `.multica` manifest and resolution

23. As a solo engineer, I want a `.multica/workspace.toml` manifest at my umbrella directory listing the workspace slug and its repos, so that the workspace composition is declared in one local, version-controllable file.
24. As a solo engineer, I want the active workspace to be resolved by walking up directories from where I am until a `.multica` is found, so that I never pass a workspace flag and resolution works offline.
25. As a solo engineer, I want resolution to fall back to the git remote of my current repo (looked up on the server) when there is no `.multica` above me, so that detached worktrees still resolve to the right workspace.
26. As a solo engineer, I want a single-repo workspace to use the same mechanism, with `.multica` placed inside the repo root listing the repo as its only member, so that there is one resolution path for both umbrella and standalone shapes.
27. As a solo engineer, I want resolution to use the single workspace automatically when only one exists, so that a one-workspace setup feels like having no workspace at all.
28. As a solo engineer, I want resolution to fall back to my last-used workspace when I am outside any repo or umbrella, so that running Claude from `~/` still picks a sensible context.
29. As a solo engineer, I want an explicit override (`--workspace` flag or `MULTICA_WORKSPACE` env) as a rarely-used escape hatch, so that I can force a context when resolution would pick the wrong one.
30. As a solo engineer, I want the manifest to pin the workspace slug, so that every re-read binds to the same server workspace instead of creating a new one each session.
31. As a solo engineer, I want the manifest to be the source of truth for workspace composition and the server to be a projection reconciled from it, so that there is one authoritative place where membership lives.
32. As a solo engineer, I want a fresh machine or a wiped database to rebuild the workspace from the manifest, so that my working environment is reproducible from the file.

### Onboarding (scan-and-suggest)

33. As a solo engineer, I want a setup skill that, finding no `.multica` above me, scans the current directory for child git repos and proposes a workspace named after the folder, so that bootstrapping a multi-repo workspace is a single confirmation.
34. As a solo engineer, I want the scan to be shallow (bounded depth) and to skip vendored/`node_modules` repos, so that it does not suggest junk.
35. As a solo engineer, I want the scan to *suggest only* — never write the manifest or create a workspace without my confirmation — so that nothing happens behind my back.
36. As a solo engineer, I want repo registration to read the remote and path from git automatically, so that adding a repo to the workspace is a "yes", not a form.
37. As a solo engineer, I want a new git repo appearing in my umbrella folder to be detected on the next session and offered for addition, so that growing the workspace is also a "yes".
38. As a solo engineer, I want a repo listed in the manifest but missing on disk to produce a warning, not a silent deletion, so that a misconfiguration forces me to look rather than losing data.

### MCP and skills

39. As a solo engineer, I want `create_issue` (MCP) to accept a `repo` argument resolved against the workspace's repo list, so that each issue is tagged with its target repo at creation.
40. As a solo engineer, I want a `list_repos` MCP read tool, so that Claude can present the repo menu and assign slices.
41. As a solo engineer, I want `create_feature` to no longer take a single `target_branch`, since branches are now derived per `(feature, repo)`.
42. As a solo engineer, I want `get_feature` to return its issues grouped by repo and by dependency layer, plus the set of PRs (one per repo), so that Claude can answer "what's the status of auth-v2?" with the full cross-repo picture.
43. As a solo engineer, I want `/to-issues` to assign each decomposed slice to a repo from the workspace's repo list and confirm the assignment with me before persisting, so that repo tagging is deliberate.
44. As a solo engineer, I want `/to-issues` to keep creating dependency edges across repos (e.g. frontend blocked by backend), so that cross-repo ordering is enforced by the scheduler.

### Daemon

45. As a solo engineer, I want the daemon to resolve an issue's repo from `issue.repo_id`, find its local path and remote, and check out the resolved feature branch in that repo, so that each agent works in the right place.
46. As a solo engineer, I want the daemon to fetch an existing remote feature branch before working, so that the second issue in a repo continues from the first issue's commits.
47. As a solo engineer, I want the task brief to include the shared-branch warning (no force-push, no history rewrite, pull --rebase before push) per repo, so that agents sharing a repo's feature branch don't destroy each other's work.

## Implementation Decisions

> Vocabulary note: this PRD uses **workspace** = context grouping repos; **repo** = a first-class git repository; **feature** = a PRD spanning repos; **issue** = a unit of work targeting one repo; **manifest** = `.multica/workspace.toml`. These terms are used verbatim in code, schema, MCP tool names, and UI labels.

### Workspace keeps its meaning and its URL — it is not collapsed to a singleton

This PRD deliberately does **not** remove workspace from the data model or the web URL. Multi-workspace stays. What changes is the *meaning*: workspace is the context that groups repos, not a tenant boundary and not a repo. The existing `[workspaceSlug]` route segment, the `X-Workspace-Slug` header, and the workspace resolution middleware are kept. The web workspace switcher remains as a secondary affordance; the primary selection mechanism is the manifest-driven resolution on the client side. This keeps churn low and preserves the ability to cherry-pick upstream.

### Schema delta (over the feature-pipeline schema)

These changes fold into the consolidated `001_init.sql` if the personal-fork schema consolidation has not merged, or ship as `ALTER`s if it has. The feature-pipeline PRD's `feature.target_branch` column is **removed** as part of this PRD — it was added by feature-pipeline but is superseded here before the product is live, so no compatibility path is kept.

- **New table `repo`:**
  - `id uuid pk default gen_random_uuid()`
  - `workspace_id uuid not null` → FK `workspace(id)` ON DELETE CASCADE
  - `name text not null` (e.g. `backend`) — human label, unique within workspace
  - `remote_url text not null` — normalized git remote; the resolution key
  - `local_path text` — path the daemon checks out into; relative to the manifest root or absolute
  - `default_branch text not null default 'main'`
  - `created_at`, `updated_at timestamptz not null default now()`
  - `UNIQUE (workspace_id, remote_url)` and `UNIQUE (workspace_id, name)`
- **`workspace.repos jsonb` is removed** — superseded by the `repo` table.
- **`issue.repo_id uuid null`** → FK `repo(id)` ON DELETE SET NULL (or RESTRICT — decided during implementation; SET NULL is preferred so deleting a repo doesn't cascade-delete issue history). NULL means the issue targets no code and the daemon skips checkout.
- **`feature.target_branch` removed.** Add **`feature.branch_slug text null`** — when set, the slug used to build `feature/<branch_slug>`; when NULL, derived from the feature's identifier/title. The branch name is identical across all repos a feature touches.
- **`github_pull_request.repo_id uuid null`** → FK `repo(id)`, so a feature's PRs can be grouped by repo. Existing `workspace_id` stays.
- **Indexes:** `repo(workspace_id)`, `issue(repo_id)`, `github_pull_request(repo_id)`.

### Deep module — manifest resolver (`internal/workspace/manifest`)

Pure logic over an injected filesystem interface. No direct OS calls in the core functions; the FS is an interface so tests use an in-memory fake.

- `Find(startDir string, fs FS) (manifestPath string, found bool)` — walk up parent directories from `startDir` until a `.multica/workspace.toml` is found or the filesystem root is reached.
- `Parse(data []byte) (Manifest, error)` — decode the TOML.
- `Manifest{ Workspace string; Repos []RepoEntry }`, `RepoEntry{ Name, Path, Remote string }`. `Path` is relative to the manifest's directory (the umbrella root).

This is the heart of the DX and the highest-value test target: total coverage is feasible because every interesting case (no manifest, manifest at root, manifest several levels up, malformed TOML, relative path resolution, single-repo manifest inside a repo) is pure data.

### Deep module — repo scanner (`internal/workspace/scan`)

- `Scan(root string, maxDepth int, fs FS, git GitRunner) []Candidate` where `Candidate{ Name, Path, Remote string }`.
- Walks children of `root` up to `maxDepth` (default shallow — 2). A directory containing `.git` is a candidate; its remote is read via the injected `GitRunner` (`git remote get-url origin`).
- Skips ignore directories: `node_modules`, `vendor`, `.git` internals, `dist`, `build`, and any directory already inside a discovered repo (do not descend into a repo to find nested repos).
- Suggest-only: the scanner returns candidates; writing the manifest and creating the workspace is a separate, confirmed step.

### Deep module — branch resolver, now per `(feature, repo)` (`internal/feature/branch`)

Evolves the feature-pipeline resolver. Pure, no DB, no I/O.

```go
type Feature struct { Identifier string; BranchSlug *string }
type Repo    struct { Name string; DefaultBranch string }
type Issue   struct { Identifier string; Metadata map[string]any }

func Resolve(i Issue, f *Feature, r *Repo) (branch string, shared bool)
```

Rules, in order:
- `issue.metadata.target_branch` set → that branch, `shared=false` (explicit override wins).
- else feature present → `feature/<BranchSlug or Identifier>`, `shared=true` (all issues of the feature in this repo converge on this branch).
- else → `issue/<Identifier>`, `shared=false`.

The branch *name* is independent of the repo (it's `feature/auth-v2` in every repo). `shared` means *multiple issues push to it within one repo*. The repo dimension matters for the **claim gate**, not for the name.

SQL mirror: the claim query implements `resolveBranch(i, f)` matching this Go function. A parity test (below) asserts SQL and Go agree on a fixture set.

### Deep module — manifest↔server reconciler (`internal/workspace/reconcile`)

Pure function over two data structures; the executor that applies the plan via the API client is separate.

- `Reconcile(m Manifest, srv WorkspaceState) Plan` where `Plan{ CreateWorkspace bool; ReposToCreate []RepoEntry; ReposMissingOnDisk []string; ReposOrphanOnDisk []Candidate }`.
- `CreateWorkspace` is true when the manifest's workspace slug does not exist on the server (fresh DB / new machine → rebuild).
- `ReposToCreate` are manifest repos absent from the server.
- `ReposMissingOnDisk` are manifest repos whose `Path` does not exist (warn, never delete).
- `ReposOrphanOnDisk` are git repos found in the umbrella but absent from the manifest (offer to add).

### Active-workspace resolution precedence (client side: MCP server + CLI)

Resolution runs in the MCP server / CLI on each session, stopping at the first match:

1. **Explicit override** — `--workspace <slug>` flag or `MULTICA_WORKSPACE` env.
2. **`.multica` walk-up** — `manifest.Find` from cwd → workspace slug + repo list (offline; the common case).
3. **git remote → server lookup** — the cwd repo's `origin` is matched against `repo.remote_url` on the server (covers detached worktrees living outside the umbrella).
4. **Single workspace** — if exactly one workspace exists, use it.
5. **Last-used** — from `~/.multica/config.toml` (mirrors the web's `last_workspace_slug` cookie) when outside any repo/umbrella.
6. **None** — trigger the scan-and-suggest onboarding flow.

A repo's `remote_url` is unique per workspace, so a repo cannot belong to two workspaces; resolution stays deterministic. When the walk-up (2) finds a manifest, it wins over the git-remote lookup (3).

### Claim handler — three gates, branch gate now per `(repo, branch)`

Extends the feature-pipeline claim query. Three `NOT EXISTS` / predicate clauses:

```sql
-- Gate 1: dependencies satisfied (blocking deps only) — unchanged from feature-pipeline
AND NOT EXISTS (
  SELECT 1 FROM issue_dependency d
  JOIN issue b ON d.depends_on_issue_id = b.id
  WHERE d.issue_id = i.id
    AND d.type IN ('blocks','blocked_by')
    AND b.status != 'done'
)

-- Gate 2: target branch not currently held by another dispatched task IN THE SAME REPO
AND NOT EXISTS (
  SELECT 1 FROM agent_task_queue t2
  JOIN issue i2 ON t2.issue_id = i2.id
  LEFT JOIN feature f2 ON i2.feature_id = f2.id
  WHERE t2.status = 'dispatched'
    AND t2.id != t.id
    AND i2.repo_id = i.repo_id                       -- same repo …
    AND resolveBranch(i2, f2) = resolveBranch(i, f)  -- … same branch
)

-- Gate 3: feature must be approved before its issues become eligible — unchanged
AND (f.status IS NULL OR f.status = 'in_progress')
```

The decisive change is `AND i2.repo_id = i.repo_id` in Gate 2: two issues sharing a branch *name* but in *different repos* (the cross-repo feature case) are no longer serialized against each other — backend and frontend run in parallel. Issues with `repo_id IS NULL` are exempt from Gate 2.

### Daemon — per-issue repo checkout and cross-repo brief

- The claim payload (`server/internal/daemon/types.go::Task`) gains, in addition to the feature-pipeline fields: `RepoName`, `RepoRemoteURL`, `RepoLocalPath`. Populated from the resolved `repo` row.
- `runTask` resolves the worktree against `RepoLocalPath` / `RepoRemoteURL` and checks out the resolved branch (existing repocache logic already fetches then checks out an existing remote branch or branches off `origin/HEAD`).
- Brief assembly (`prompt.go`):
  - The shared-branch section (from feature-pipeline) is emitted when the resolved branch is `shared`, scoped to the repo.
  - A new **cross-repo context** section is appended when the issue's feature touches more than one repo: a summary of sibling issues, the repo each targets, and any cross-repo contracts supplied as text (e.g. the API schema). **v1 default is text-context.** Mounting read-only sibling checkouts is a follow-up (see Out of Scope).

### MCP and skills

- **`create_issue`** gains a `repo` parameter (name or id) resolved against the active workspace's repos; optional (NULL allowed for coordination issues).
- **`create_feature`** drops `target_branch`.
- **`list_repos`** new read tool — returns the workspace's repos so the model can present the assignment menu.
- **`get_feature`** returns child issues grouped by repo and by dependency layer, plus the PR set (one per repo) resolved via `issue_pull_request` joined to `github_pull_request.repo_id`.
- **Setup skill** (`.claude/skills/`): on no-manifest, run the scanner, present candidates, on confirmation write `.multica/workspace.toml`, create the workspace, register repos.
- **`/to-issues` override**: assign each slice to a repo from `list_repos`, confirm assignments and the dependency graph, then `create_issue` per slice with `repo` and `feature_id` set, then `link_issue_dependency` for cross-repo edges.
- **`/to-prd` override**: unchanged except it no longer passes a target branch.

### Manifest format and location

- `.multica/workspace.toml` (directory form), consistent with the existing `~/.multica/config.toml` CLI config convention and leaving room for local cache/state alongside it later.
- Content shape:

```toml
workspace = "meu-produto"     # stable slug, pinned at creation; binds re-reads to the same server workspace

[[repo]]
name   = "backend"
path   = "./backend"          # relative to this manifest's directory (the umbrella root)
remote = "github.com/voce/backend"

[[repo]]
name   = "frontend"
path   = "./frontend"
remote = "github.com/voce/frontend"

[[repo]]
name   = "qa"
path   = "./qa"
remote = "github.com/voce/qa"
```

- The umbrella directory is typically **not** itself a git repo, so the manifest is a local, uncommitted file by default. It is intentionally portable: copying/cloning the umbrella and running a session rebuilds the workspace via the reconciler. For a single-repo workspace, the manifest lives at the repo root listing the repo as its only member.

### Web dashboard adjustments

- The feature detail page groups child issues by **repo**, then by dependency layer (ready / blocked), reusing the existing grouping introduced by feature-pipeline.
- The header shows the feature branch name and the **set of PRs (one per repo)** with their statuses, instead of a single PR link.
- No new pages; the repo grouping and the multi-PR header are rearrangements of the feature-pipeline feature view.

## Testing Decisions

A good test here asserts **external behavior, not internal structure**: pure modules are exercised as table-driven unit tests over data; the claim gate is exercised via real HTTP/DB integration; nothing reaches into private helpers. Per the user's direction, **all four deep modules get tests**, plus the scheduler integration tests.

### Modules that get new tests

- **Manifest resolver** (`internal/workspace/manifest`). Table-driven over an in-memory FS. Cases: no manifest anywhere → not found; manifest in cwd; manifest several levels up (walk-up); manifest at filesystem root boundary; malformed TOML → error; relative repo paths resolved against the manifest dir; single-repo manifest inside a repo root.
- **Repo scanner** (`internal/workspace/scan`). Table-driven over an in-memory FS + fake `GitRunner`. Cases: flat umbrella with 3 child repos → 3 candidates; depth bound respected (repo at depth 3 not found when maxDepth=2); `node_modules`/`vendor` skipped; no descent into a discovered repo to find nested repos; remote read via the fake git runner; a child dir with no `.git` ignored.
- **Branch resolver** (`internal/feature/branch`). Table-driven, pure. Cases: issue override set → override branch, shared=false; feature present, no override → `feature/<slug>`, shared=true; feature `BranchSlug` set vs derived from `Identifier`; no feature → `issue/<id>`, shared=false; identical branch name across two different repos (name independent of repo).
- **SQL/Go branch parity** (`internal/feature/branch_parity_test.go`). Asserts the SQL `resolveBranch` and the Go `Resolve` produce identical results for a fixture set; catches drift if one side is edited alone.
- **Reconciler** (`internal/workspace/reconcile`). Table-driven, pure. Cases: workspace absent on server → `CreateWorkspace=true`; manifest repo absent on server → in `ReposToCreate`; manifest repo path missing on disk → in `ReposMissingOnDisk`; orphan git repo on disk not in manifest → in `ReposOrphanOnDisk`; fully in-sync manifest → empty plan.
- **Claim gate — cross-repo branch serialization** (extends `server/internal/handler/daemon_test.go`). Cases: two issues sharing a feature, different repos, same branch name → both claimable in parallel; two issues sharing a feature *and* repo, same branch → first claimed, second blocked until first completes; issue with `repo_id IS NULL` → exempt from Gate 2; cross-repo dependency (frontend blocked by backend) → frontend not claimed until backend done; feature status gate unchanged.
- **Repo CRUD handler** (new, mirrors existing handler tests). Create repo under workspace; unique `(workspace_id, remote_url)` and `(workspace_id, name)` enforced; list repos for workspace; `issue.repo_id` set on create and returned.

### Modules whose tests get adapted

- Feature-pipeline branch resolver tests are rewritten for the three-argument signature.
- Feature-pipeline claim-handler branch-gate tests gain the `repo_id` dimension.
- `get_feature` MCP tool test asserts the new grouped-by-repo + multi-PR response shape.

### Prior art

- `server/internal/handler/issue_test.go` and `daemon_test.go` — the established handler/scheduler integration pattern (test DB, `httptest`, assert on response and DB state). The claim and repo-CRUD tests follow this exactly.
- `packages/core/i18n/pick-locale.test.ts` — the model for pure table-driven tests (the branch/manifest/scanner/reconciler modules follow this structure, in Go).
- The feature-pipeline PRD's branch resolver and its parity test are the direct ancestors of the three-argument resolver and its parity test here.

## Out of Scope

- **Collapsing workspace to a singleton or removing it from the URL.** Multi-workspace stays; this PRD redefines its meaning, it does not delete it.
- **Mounting read-only sibling-repo checkouts in the agent worktree.** v1 cross-repo context is text only (contracts injected into the brief). Read-only sibling mounts are a follow-up.
- **A web UI for creating repos, features, or issues.** Creation stays MCP-only (inherited from feature-pipeline). Repo registration happens via the setup skill / reconciler, not a web form.
- **Committing the `.multica` manifest to a meta-repo / syncing it across machines automatically.** The manifest is a local file; portability is by copy, not by a sync service.
- **Cross-repo atomic merge.** Each repo's PR merges independently; Multica does not coordinate a simultaneous merge across the N PRs of a feature.
- **Stacked branches across repos** (repo B's feature branch based on repo A's). Each repo's feature branch is based on that repo's default branch.
- **Dependency cycle detection at link time** (inherited from feature-pipeline — still out).
- **Migrating existing `workspace.repos jsonb` data.** Pre-release; the DB is wiped and rebuilt from manifests via the reconciler.
- **Auto-detecting which repo a slice belongs to without confirmation.** `/to-issues` proposes, the user confirms. No silent repo assignment.
- **Remote/HTTP MCP transport** (inherited from feature-pipeline — still stdio only).

## Further Notes

### Relationship to the feature-pipeline PRD

This PRD is an **amendment layered on top of feature-pipeline**, not a replacement. Everything feature-pipeline specifies stands except the four amended decisions listed in the header (single target branch → per-`(feature, repo)`; resolver signature; per-`(repo, branch)` gate; one PR → N PRs). Implement feature-pipeline first; the single-repo behavior it ships is the degenerate case of this PRD (a feature touching exactly one repo behaves identically).

### Recommended order of execution

1. **`repo` table + `issue.repo_id` + `github_pull_request.repo_id`; remove `feature.target_branch`, add `feature.branch_slug`.** Schema first; repo CRUD handler + tests. No behavior change to the scheduler yet.
2. **Branch resolver three-argument rewrite + parity test.** Pure module; the scheduler imports it next.
3. **Claim gate `repo_id` dimension.** The substantive scheduler change; integration tests against a real DB.
4. **Daemon per-issue repo checkout + cross-repo brief (text context).** Self-contained; repocache unchanged.
5. **Manifest resolver + repo scanner + reconciler modules.** Pure, fully unit-tested; no wiring yet.
6. **CLI/MCP resolution precedence wired to the manifest modules; `list_repos`, `create_issue` repo arg, `create_feature` drops target branch, `get_feature` grouped response.**
7. **Setup skill (scan-and-suggest) + `/to-issues` repo tagging override.**
8. **Web feature view: group by repo, multi-PR header.** Parallel to the MCP work.

### Risk and verification

- **Resolution picking the wrong workspace.** Mitigation: deterministic precedence with the manifest winning over the git-remote lookup; `remote_url` unique per workspace; the resolver modules are fully unit-tested over fake filesystems.
- **SQL/Go branch resolver drift** (now with the repo dimension). Mitigation: the parity test, extended with repo fixtures.
- **Scanner suggesting junk.** Mitigation: bounded depth, ignore-list, no descent into discovered repos, suggest-only — all covered by scanner tests.
- **Manifest/server divergence.** Mitigation: the reconciler is pure and tested; reconciliation warns on missing repos and never deletes; the manifest is authoritative so the server is always rebuildable.
- **Cross-repo parallelism regressions.** Mitigation: the claim-gate integration tests assert both the parallel (different repos) and serial (same repo, same branch) cases explicitly.

## Comments
