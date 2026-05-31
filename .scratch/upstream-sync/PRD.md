# Upstream Sync & Workspace In-Place Execution

This file holds **two PRDs** so that the issues derived from both land in one place:

- **PRD 1 — Upstream Sync:** port a curated set of independently-valuable features and fixes from PR #2 (`juanpinheiro/multica#2`) into the personal fork.
- **PRD 2 — Workspace In-Place Execution Mode:** a new, opt-in execution mode where an agent runs in the workspace's real umbrella directory (all repos visible) instead of an isolated per-repo worktree. Amends `.scratch/multi-repo-features/PRD.md`. Ports upstream's path-safety primitives as its foundation.

PRD 2 is the design-heavy, fork-original piece; PRD 1 is a grab-bag of catch-up improvements. They are independent except that both can ship into the same post-multi-repo fork.

---

# PRD 1 — Upstream Sync: port selected upstream features & fixes

**Status:** `ready-for-agent`
**Owner:** Juan Pinheiro
**Created:** 2026-05-29
**Depends on:** `.scratch/multi-repo-features/PRD.md` (and transitively `feature-pipeline`, `multica-personal-fork`) — every port is adapted to the post-cut shape: `project` renamed to `feature`, the first-class `repo` table, and loopback + singleton auth.

## Problem Statement

After three deliberate cuts (`multica-personal-fork`, `feature-pipeline`, `multi-repo-features`), the fork diverged from upstream Multica at a fixed point and upstream kept shipping. PR #2 ("multica to local") is, despite its title, a whole-upstream sync opened against the stripped fork — ~499 files, dominated by *re-adding* everything the fork deleted on purpose (desktop, mobile, the docs site with Korean/Chinese translations, billing, helm, Cloud-Node/Fleet auth, self-hosting). Merging it would revert the personal fork wholesale.

But inside its ~60 upstream commits are genuine improvements that land squarely in surface the fork **kept**: agent CLI backends, the local daemon, autopilots, issues/board, comments, the editor, the CLI, and the views layer. A spot-check confirmed the most valuable of these are absent locally (the fork predates them). Doing nothing means the fork silently falls behind on agent-runtime support and accumulates already-fixed bugs in the exact features the owner uses daily — including runtime issues specific to the owner's Windows/WSL2 setup.

The risk in porting carelessly is dragging back a deleted concept: upstream's `project` vocabulary (renamed to `feature`), its personal-access-token / Cloud-PAT / Fleet auth (replaced by loopback + singleton), or its self-host/billing surface. Each port must be translated to the fork's shape, not copied.

## Solution

Port a curated subset of PR #2 via surgical re-implementation (never a PR merge), organized into three workstreams of independently-valuable items. Each item delivers value on its own and is translated to the fork's vocabulary and auth model as it lands.

- **Workstream A — Agent runtimes & daemon reliability:** a new agent runtime backend (Antigravity); detection of a Codex Desktop bundle CLI; retry of terminal task callbacks on transient errors; a guard against garbage-collection metadata with empty parent ids; consolidation of a WSL2 local daemon under the single `LOCAL` machine instead of a separate out-of-band runtime.
- **Workstream B — Autopilots, issues, comments, CLI:** per-trigger webhook event filters for autopilots; trigger-output timezone correctness with an invalid-zone fallback; preservation of an issue's parent through the create-with-agent flow; triggering a parent issue's agent assignee when a child issue completes; clearing deleted ids from the recent-issues client store; placing a newly created issue at the top of its column in manual sort mode; multi-attachment comments and edit-time attachment removal; roots-only comment listing in the CLI and removal of a noisy stderr preamble.
- **Workstream C — Views & editor quality:** unifying detail/list page headers into one shared breadcrumb header; a React-hygiene cleanup sweep; a board fix for empty swimlanes under pagination; a sticky live-agent bar on the issue view; an editor fix to render code blocks when auto-highlight yields an empty tree; an editor fix to preserve raw html-like text on paste. Optionally, an agent-level MCP configuration tab.

Everything else in PR #2 is explicitly out of scope (see the skip list).

## User Stories

1. As the fork owner, I want the Antigravity agent CLI recognized as a runtime backend, so that I can assign issues to it like any other local agent.
2. As the fork owner, I want a new agent backend to register through the same mechanism every existing backend uses, so that adding it carries no special-case wiring.
3. As the fork owner on Windows, I want my WSL2 local daemon consolidated under the `LOCAL` machine, so that I see one local runtime instead of a confusing out-of-band duplicate.
4. As the fork owner, I want the daemon to consolidate local machines by device name, so that re-registrations don't fan out into phantom runtimes.
5. As the fork owner, I want the daemon to detect a Codex Desktop bundle CLI automatically, so that Codex works without me hand-wiring its path.
6. As the fork owner, I want terminal task callbacks retried on transient errors, so that a network blip doesn't strand a finished task in a non-terminal state.
7. As the fork owner, I want the daemon to ignore garbage-collection metadata with empty parent ids, so that a malformed record can't crash or corrupt the GC pass.
8. As the fork owner, I want autopilot webhook triggers to filter by event type per trigger, so that one webhook source can drive different autopilots for different events.
9. As the fork owner, I want autopilot trigger output rendered in the trigger's own timezone, so that scheduled output reads correctly regardless of server timezone.
10. As the fork owner, I want a sane default when a trigger's timezone is invalid, so that a typo in a zone name degrades gracefully instead of breaking the trigger.
11. As the fork owner, I want a sub-issue's parent preserved when I create it through the create-with-agent flow, so that the hierarchy I intended survives.
12. As the fork owner, I want completing a child issue to trigger the parent issue's agent assignee, so that multi-step agent work advances without me poking it.
13. As the fork owner, I want deleted issue ids cleared from the recent-issues list, so that the recents don't surface tombstones I can't open.
14. As the fork owner, I want a newly created issue placed at the top of its column in manual sort mode, so that new work appears where I expect it.
15. As the fork owner, I want to attach multiple files to a comment, so that an agent or I can carry several pieces of evidence in one comment.
16. As the fork owner, I want to remove an attachment while editing a comment, so that I can correct a mistaken upload without deleting the whole comment.
17. As the fork owner, I want `multica issue comment list` to show roots-only by default, so that the CLI (and the MCP server that wraps it) returns clean, thread-rooted output.
18. As the fork owner, I want the CLI to drop its "Showing N comments." stderr preamble, so that piped/scripted output is not polluted by chatter.
19. As the fork owner, I want detail and list pages to share one breadcrumb header, so that headers behave consistently and there is one place to change them.
20. As the fork owner, I want a React-hygiene cleanup (correct button types, modern context consumption, non-mutating sorts, error-boundary fixes), so that the views layer carries fewer latent bugs.
21. As the fork owner, I want the board's empty swimlanes to render correctly under pagination, so that lanes don't silently disappear.
22. As the fork owner, I want a sticky live-agent bar on the issue view, so that I can see at a glance which agents are running on the issue I'm looking at.
23. As the fork owner, I want the editor to render code blocks even when auto-highlight returns an empty tree, so that fenced code stops vanishing.
24. As the fork owner, I want the editor to preserve raw html-like text on paste, so that pasting markup doesn't eat my content.
25. As the fork owner, I optionally want an agent-level MCP configuration tab, so that an agent can be pointed at MCP servers from its detail page.
26. As the fork owner, I do NOT want Cloud-Node PATs, Fleet auth, cloud billing, the helm chart, self-hosting docs, Korean/Chinese docs, desktop, or mobile, because the fork removed all of those on purpose.

## Implementation Decisions

> Translation rule for every port: upstream code that says `project` / `project_resource` becomes `feature` / `feature_resource`; upstream auth helpers tied to personal-access-tokens, Cloud-PATs, or Fleet are dropped, because the fork attributes every request to the singleton user via the loopback middleware. Because the rename guarantees conflicts in most server files, server-side ports are **re-implemented against the fork's names**, not force-cherry-picked.

### Workstream A — Agent runtimes & daemon reliability

- **Antigravity runtime backend.** Add a new agent backend implementing the same interface as the fork's existing backends (Claude, Codex, Gemini, Hermes, Kimi, Kiro, OpenClaw, OpenCode, Pi, Cursor, Copilot) and register it in the agent model registry. Pure addition; no coupling to any deleted surface. The Antigravity entry is fit into the fork's registry shape, which may already differ from upstream — port the backend into the fork's structure, not the reverse.
- **Codex Desktop bundle CLI detection.** Extend the daemon's Codex CLI discovery to recognize a Codex Desktop bundle location. Self-contained path-probe addition.
- **Terminal task callback retries.** Wrap the daemon's terminal callback (the task→server result report) in a bounded retry on transient (network / 5xx) failures, so a finished task isn't stranded by a momentary failure to report.
- **GC empty-parent-id guard.** In the daemon's garbage-collection pass, skip metadata rows whose parent ids are empty rather than acting on them. Pure defensive fix.
- **WSL2 local-daemon consolidation.** Consolidate a WSL2-hosted daemon under the single `LOCAL` machine and de-duplicate local machines by device name, so the runtimes list shows one local entry rather than an out-of-band split. Verified against the fork's runtime/heartbeat model, which the fork kept.

### Workstream B — Autopilots, issues, comments, CLI

- **Autopilot per-trigger webhook event filters.** Add the ability to filter, per webhook trigger, which event types fire the autopilot. The schema change folds into the fork's consolidated init rather than shipping as a standalone migration, consistent with the fork's one-init decision. The matching logic (does an incoming event match a trigger's filter?) is extracted as a pure module so it can be tested in isolation. Non-matching events are recorded as skips through the existing autopilot-run audit, not silently dropped.
- **Autopilot timezone correctness.** Render trigger output in the trigger's configured timezone, centralize the default timezone, and fall back to that default when a configured zone is invalid.
- **Preserve parent through create-with-agent.** Ensure the issue's parent id survives the create-with-agent code path so sub-issue hierarchy is not lost.
- **Child-done triggers parent's agent assignee.** When a child issue transitions to done, trigger the parent issue's agent assignee, advancing multi-step agent work.
- **Recent-issues store cleanup.** Clear deleted issue ids from the client-side recent-issues store so the list cannot surface tombstones.
- **New issue at top in manual sort.** When manual sort mode is active, position a newly created issue at the top of its column.
- **Comment attachments.** Support selecting multiple attachments on a comment and removing an attachment while editing. The unstable since-delta / comment-session-resume work from the same PR is explicitly excluded (it was reverted twice upstream).
- **CLI comment listing.** Default the CLI comment list to roots-only output and remove the "Showing N comments." stderr preamble, producing clean output for both human use and the MCP server that wraps the CLI.

### Workstream C — Views & editor quality

- **Shared breadcrumb header.** Unify the detail-page and list-page headers into one shared breadcrumb header component, removing duplication. Lands late so it rebases cleanly over the other view changes.
- **React-hygiene cleanup.** Apply the upstream React cleanup (button types, modern context consumption, non-mutating array sorts, error-boundary fixes) selectively to files the fork still has; skip anything that only existed for deleted platforms.
- **Swimlane empty-lane fix.** Fix the board so empty swimlanes render correctly under pagination.
- **Sticky live-agent bar.** Add the sticky live-agent bar to the issue view, porting the net final upstream state (after the follow-up that adjusted its placement), not both revisions.
- **Editor fixes.** Render code blocks when auto-highlight returns an empty tree; preserve raw html-like text on paste. Two self-contained editor correctness fixes.
- **Optional agent MCP-config tab.** A tab on the agent detail page for configuring MCP servers the agent can consume (including ACP runtimes). Lowest priority; included only if the owner wants per-agent MCP config now. Distinct from the fork's own `multica mcp` server.

### Skipped (from PR #2)

Cloud-Node PATs / Fleet auth and renewal; cloud billing proxy + Stripe webhook + billing test page; helm chart publish; release pipeline; the self-host workspace-creation toggle and SMTP/Exchange docs; all docs-site content including Korean/Chinese translations; desktop and mobile apps and the mobile CI split. These re-introduce concepts removed across the three prior PRDs.

## Testing Decisions

A good test here asserts external behavior, not implementation detail: backend handlers exercised over real HTTP against a test database; pure helpers as table-driven unit tests; views via rendering + simulated interaction. Coverage concentrates on new logic, not on the items that are pure deletions or mechanical renames.

- **Autopilot event-filter matcher** (pure module). Table-driven: an event whose type is in a trigger's filter fires; an event outside the filter is skipped and recorded; an empty/absent filter preserves prior fire-on-everything behavior. Prior art: the existing autopilot webhook handler tests.
- **Child-done parent trigger** (handler/integration). Completing a child issue triggers the parent's agent assignee; completing a child with no agent-assigned parent is a no-op. Prior art: the existing child-done handler tests.
- **Antigravity backend** (unit). Argv construction, version probe, and model mapping, mirroring an existing backend's test.
- **Adapted ports.** Any ported server test is renamed `project`→`feature` and stripped of Cloud-PAT/Fleet setup before it compiles. Comment-attachment tests keep the multi-attachment and edit-time-removal cases and drop any since-delta cases.
- **Confirm-absent before porting.** For each Workstream B/C fix, the fork is grepped first; an item already present from the fork base is skipped rather than double-applied.
- **Not tested:** the skipped items (no code); view refactors rely on existing view tests as the regression net unless a fix changes behavior.

## Out of Scope

- Merging PR #2 or tracking upstream continuously — this is a one-time, curated port.
- Re-adding any deleted platform or SaaS surface (see the skip list).
- The since-delta comment machinery (reverted twice upstream).
- In-place execution — that is PRD 2 below, not part of this catch-up.

## Further Notes

- **Order:** Workstream A first (additive, low-risk, immediate value), then B (independent items in any order; the event-filter schema folds into the consolidated init), then C (header unification and the React sweep last so they rebase cleanly).
- **Provenance:** all items trace to commits on PR #2 (head `973a4392`). Exact upstream commit references live in the implementation issues, not here, since file paths drift.

---

# PRD 2 — Workspace In-Place Execution Mode

**Status:** `ready-for-agent`
**Owner:** Juan Pinheiro
**Created:** 2026-05-29
**Amends:** `.scratch/multi-repo-features/PRD.md` — adds a second execution mode alongside the worktree model; the per-`(repo, branch)` claim gate and the per-`(feature, repo)` branch derivation are unchanged for worktree repos and bypassed for in-place workspaces as described below.
**Depends on:** `.scratch/multi-repo-features/PRD.md` (the `repo` table, the `.multica/workspace.toml` manifest, and walk-up resolution must exist). Ports upstream's path-safety primitives from PR #2 as its foundation.

## Problem Statement

The multi-repo pipeline runs every agent task in an isolated git worktree the daemon cuts under its own cache root: the daemon resolves the issue's repo, creates a worktree on a derived `feature/<slug>` branch, the agent works there, and the work converges into one PR per `(feature, repo)`. Isolation is deliberate — agents never touch the developer's real working tree, and parallel feature branches stay clean.

But the developer's actual mental model, and the way they already use agent CLIs, is different: they open the agent **at the umbrella directory** that holds all their repos as children (`~/code/meu-produto/` with `backend/`, `frontend/`, `qa/` inside), so the agent can see and reason across every repo at once. The worktree model can't express this — it scopes each task to a single repo's isolated checkout, so a cross-repo feature is split into per-repo worktrees that are blind to each other. The agent working the frontend slice can't `grep` the backend it depends on; it only gets whatever cross-repo context was injected as text.

There is no way today to say "for this workspace, run the agent in my real umbrella directory, with all repos visible, the way I'd run it by hand." And the safety machinery that running in a real user directory demands — refusing to operate on a system root or `$HOME`, serializing two tasks that would fight over the same on-disk tree, surfacing a clear waiting state instead of a silently stuck task — does not exist in the fork. Upstream built exactly those primitives for a related (per-directory) feature; the fork can harvest them.

## Solution

Add a second, opt-in **execution mode**, selected per workspace in the manifest, that runs the agent in the workspace's real umbrella directory with every repo visible as a child — instead of an isolated per-repo worktree. The mode is the developer's lever, and choosing it is an informed trade: in-place execution is **serial per workspace** (a single real working tree can't host two concurrent agents without corruption), whereas worktree mode stays parallel across repos. The developer opts in knowing they trade parallelism for whole-workspace visibility and for working in their real tree.

Concretely:

- **`mode` in the manifest.** The workspace manifest gains a top-level `mode` of `worktree` (default) or `in_place`. It is the only place the mode is set — no CLI flag, no dashboard form. The reconciler projects it onto the server's workspace record.

```toml
# .multica/workspace.toml
workspace = "meu-produto"
mode = "in_place"          # default is "worktree"; in_place runs serially in the umbrella
[[repo]] name = "backend"  path = "./backend"  remote = "github.com/voce/backend"
[[repo]] name = "frontend" path = "./frontend" remote = "github.com/voce/frontend"
```

- **In-place runs at the umbrella root.** When a workspace is `in_place`, the agent's working directory is the umbrella directory (where `.multica/workspace.toml` lives), with all declared repos as children. The agent has read/write visibility across every repo simultaneously.
- **The branch/PR flow is preserved, not bypassed.** This is the key constraint: the agent flow does not change. Before the run, the daemon prepares each repo the task touches by checking out (or creating) that repo's `feature/<slug>` branch in the **real** repo, exactly as the worktree mode derives it. The agent works across the umbrella, commits per repo, and the convergence into one PR per repo is identical to the multi-repo PRD. In-place changes **where** the agent runs and **what it can see**, not how branches and PRs are formed.
- **Fail-fast over clobber.** Because the daemon now switches branches in the developer's real repos, it refuses to proceed when a touched repo is dirty or not on an expected branch, failing the task with a clear message rather than overwriting in-progress work. This mirrors upstream's "fail fast rather than guess" stance for operating on a real directory.
- **Safety primitives, ported from upstream.** Three reusable pieces from PR #2's local-directory work are ported as the foundation: a path validator that refuses system roots, the user's home, and Windows drive roots (checked literally and after resolving symlinks, with a real read/write probe); a path locker that serializes tasks sharing an on-disk directory; and a new `waiting_local_directory` task status with a wait-reason hint. In this PRD the locker keys on the **umbrella** directory, so a second task in an in-place workspace parks in `waiting_local_directory` until the first releases — making the serial trade explicit and observable rather than a silent stall.
- **The mode is visible in the UI.** Every place a run is surfaced (the runtime/task view and the feature view) shows whether it is executing in `worktree` or `in_place` mode, so the developer always knows which model is in effect and why a second task is waiting.
- **`.multica` is shared intentionally.** Because the in-place working directory is the umbrella, the daemon's runtime context and the workspace manifest both live under the umbrella's `.multica/` — the manifest as `.multica/workspace.toml`, the runtime context under a distinct child path. They never collide (different names/subpaths); the reconciler only ever reads the manifest. This is documented so the two never drift.

## User Stories

1. As the fork owner, I want to run an agent in my workspace's real umbrella directory with all repos visible, so that a cross-repo task can read and edit backend, frontend, and qa at once.
2. As the fork owner, I want in-place mode to be opt-in per workspace, so that workspaces I haven't configured keep the isolated worktree behavior.
3. As the fork owner, I want to set the mode only in the manifest, so that the workspace's execution model is declared in one local, version-controllable place.
4. As the fork owner, I want `worktree` to be the default, so that nothing changes for a workspace until I explicitly choose in-place.
5. As the fork owner, I want choosing in-place to be a deliberate trade I control, so that I accept serial execution in exchange for whole-workspace visibility.
6. As the fork owner, I want to know that in-place execution is serial per workspace, so that I'm not surprised when a second task waits instead of running in parallel.
7. As the fork owner, I want worktree workspaces to keep running repos in parallel, so that I don't lose throughput where isolation is fine.
8. As the fork owner, I want a second task in an in-place workspace to enter a clear waiting state with a reason, so that the dashboard never shows it silently stuck.
9. As the fork owner, I want the branch and PR flow to stay exactly as it is in worktree mode, so that switching a workspace to in-place doesn't break how my agents produce branches and PRs.
10. As the fork owner, I want the daemon to put each touched repo on its `feature/<slug>` branch before the run, so that in-place work still converges into one PR per repo.
11. As the fork owner, I want the daemon to refuse to run when a repo is dirty or off its expected branch, so that an agent never switches branches over my in-progress work.
12. As the fork owner, I want the daemon to refuse to run an agent against a system root, my home directory, or a Windows drive root, so that a misconfigured path can't make an agent write across my whole machine.
13. As the fork owner on Windows, I want drive roots and UNC paths rejected as in-place targets, so that the safety check is correct on my platform, not just on POSIX.
14. As the fork owner, I want symlinked paths that resolve to a banned location also rejected, so that a symlink can't slip a dangerous target past the check.
15. As the fork owner, I want two tasks that would share the same real directory serialized, so that two agents never corrupt each other's working tree.
16. As the fork owner, I want the umbrella's runtime context and my workspace manifest to coexist under `.multica` without colliding, so that running an agent in the umbrella doesn't fight with workspace resolution.
17. As the fork owner, I want every run to show whether it is in `worktree` or `in_place` mode in the UI, so that the execution model is always visible to me.
18. As the fork owner, I want the feature view to indicate when its workspace runs in-place, so that I understand why its issues run serially.
19. As the fork owner, I want a fresh machine or wiped database to rebuild the mode from the manifest, so that my execution model is reproducible from the file like the rest of the workspace.
20. As the fork owner, I want my own `CLAUDE.md` / `AGENTS.md` / `GEMINI.md` preserved during in-place runs, so that the daemon doesn't clobber my instructions with generated context.
21. As the fork owner, I want an in-place agent to see sibling repos directly rather than only as injected text context, so that cross-repo contracts can be read from source.
22. As the fork owner, I do NOT want a web form for setting the mode, so that the manifest stays the single source of truth.

## Implementation Decisions

> Vocabulary: **workspace** = the context grouping repos (and the umbrella directory it anchors to); **repo** = a first-class git repository under the umbrella; **feature** = a PRD spanning repos; **manifest** = `.multica/workspace.toml`. **Execution mode** is a new per-workspace attribute. All terms are used verbatim in code, schema, and UI.

### Mode as a workspace attribute, manifest-driven

- The manifest gains a top-level `mode` (`worktree` | `in_place`), default `worktree`. The workspace record gains a corresponding mode attribute, folded into the consolidated init schema. The manifest is authoritative; the reconciler sets the server's mode from it on session start, so a wiped database rebuilds the mode from the file.
- No CLI flag and no dashboard control set the mode. The setup/reconcile flow surfaces, at the moment a workspace is created or switched to in-place, a note that in-place execution is serial — making the trade explicit at the point of choice.

### Execution-mode resolution (deep module)

A pure decision that, given the workspace mode, the resolved repo, and the issue, returns how and where to run: for a worktree workspace, the existing behavior (isolated worktree + derived `feature/<slug>` branch, parallel-eligible); for an in-place workspace, the umbrella directory as the working directory with the same `feature/<slug>` branch prepared in the real repo, serial-eligible. This isolates the branching decision from any I/O so it can be unit-tested over data, and extends — rather than replaces — the multi-repo branch resolver.

### Path validator (deep module, ported)

A pure, OS-aware validator that rejects, for an in-place target: non-absolute paths; system roots, the user's home, and Windows drive roots / UNC roots; any of those reached through a symlink (checked both literally and after symlink resolution); paths that don't exist, aren't directories, or aren't read/write (verified by a transient probe). Ported close to verbatim from upstream because it is security-critical and the Windows handling matters for the owner. It validates the umbrella directory before an in-place run.

### Path locker (deep module, ported)

A per-realpath lock, held for a task's whole lifetime (claim → context write → agent run → report), keyed on the symlink-resolved umbrella path so two routes to the same directory collapse to one lock. Acquiring a held lock fires a one-shot callback (used to flip the task into the waiting state) and then blocks until the lock frees or the context is cancelled; release is idempotent and cancellation never wedges a future acquirer. Ported close to verbatim — it has no `project`/`feature` coupling. For in-place workspaces it is the serialization mechanism, in place of the worktree mode's per-`(repo, branch)` claim gate.

### Waiting task status

A new `waiting_local_directory` task status plus a wait-reason hint, folded into the consolidated init schema (not a standalone migration). The daemon posts it via the locker's wait callback when an in-place task finds the umbrella busy, with a reason naming the held path and holder; it clears to `running` once the lock is acquired. The runtimes/task UI renders it.

### Daemon: in-place preparation and the preserved flow

For an in-place workspace, the daemon validates the umbrella path, acquires the umbrella lock (parking as waiting if held), and prepares each touched repo by checking out or creating its `feature/<slug>` branch in the real repo — failing fast if the repo is dirty or off an expected branch. The agent then runs at the umbrella root. Commit, push, and one-PR-per-repo convergence are unchanged from the multi-repo PRD. The user's own instruction files are preserved rather than overwritten by generated context. When the workspace is `worktree`, none of this engages and behavior is exactly as today.

### UI: execution-mode indicator

The runtime/task view and the feature view display the active execution mode (`worktree` or `in_place`) for each run, and the waiting state renders its reason. The indicator is a read-only surfacing of server state — no new control, consistent with the manifest being the only place the mode is set.

### `.multica` namespacing

The umbrella's `.multica/` holds both the manifest (`workspace.toml`) and the daemon's in-place runtime context under a distinct child path; they share the directory by design and never clobber. The reconciler and walk-up resolution read only the manifest. This convention is documented in the project guide so the manifest and runtime cache never drift into a conflict.

## Testing Decisions

External-behavior tests over the same patterns the fork already uses: pure modules as table-driven unit tests; the daemon/claim behavior as integration tests against a real database; the UI indicator via rendering. The four pieces with the highest value-to-size ratio are the deep modules.

- **Path validator** (pure, ported tests adapted). Absolute-path required; system-root, home, and Windows drive-root / UNC rejection; symlink targets that resolve to a banned path rejected; non-existent / non-directory / non-writable rejected; a legitimate project directory accepted. The Windows cases are required and must run on the owner's machine. Highest-value target.
- **Path locker** (concurrency). Fast-path acquire; a second acquirer on the same realpath blocks and fires the wait callback once with the holder; release frees the next waiter; cancellation while waiting returns without taking the lock and doesn't wedge a future acquirer; distinct realpaths don't contend; symlink-aliased paths collapse to one lock.
- **Execution-mode resolution** (pure). Worktree workspace → isolated worktree + derived branch, parallel-eligible; in-place workspace → umbrella working directory + same derived branch prepared in the real repo, serial-eligible. Extends the multi-repo branch-resolver tests.
- **Waiting-status flow** (integration). A second in-place task in a busy workspace transitions to `waiting_local_directory` with a populated reason, then to `running` once the first releases; two tasks in distinct worktree repos run in parallel (the gate lets both, the locker doesn't engage). This is the test that guards against the gate and the locker double-blocking or deadlocking.
- **Fail-fast preparation** (integration). An in-place task against a dirty or off-branch real repo fails with a clear message and does not switch branches.
- **UI mode indicator** (view). A run renders its mode; a waiting task renders its reason.
- **Not tested:** the manifest `.multica` namespacing is a convention, asserted implicitly by the multi-repo reconciler tests continuing to pass once the daemon writes its in-place context under the umbrella's `.multica`.

## Out of Scope

- Removing or weakening the worktree mode — it stays the default and is unchanged.
- Per-repo or per-issue mode granularity — the mode is a workspace attribute only. Mixed modes within one workspace are not supported in v1.
- A CLI flag or dashboard control for the mode — manifest only.
- Parallel in-place execution within one workspace — in-place is serial by design; the locker enforces it.
- Mounting read-only sibling checkouts in a worktree (the multi-repo follow-up) — superseded for in-place workspaces, which already expose all repos; still out for worktree workspaces.
- An in-place mode where the agent manages branches itself with the daemon not touching git — considered (the "even simpler" option) and deferred; v1 keeps the daemon preparing branches so the existing agent flow is preserved.
- Cross-repo atomic merge or coordinating the N PRs of a feature — unchanged from multi-repo; each repo's PR merges independently.

## Further Notes

- **Why workspace-level, not per-repo:** the developer's working model is one agent session at the umbrella seeing everything, which is also the cross-repo visibility the multi-repo PRD wanted but could only approximate with injected text context. Per-repo in-place would reintroduce the blindness without buying anything the worktree mode doesn't already give.
- **Why the flow is preserved:** the owner's explicit constraint is not to break the agent flow. Keeping daemon-side branch preparation means in-place is purely a change of location and visibility; branches, commits, and PR-per-repo convergence are identical to worktree mode, so a workspace can flip between modes without the downstream pipeline noticing.
- **Order of execution:** fold the `mode` attribute and the `waiting_local_directory` status into the consolidated init; port the path validator and locker as standalone tested modules; add the execution-mode resolution module; wire the daemon (validate → lock → prepare branches → run at umbrella → preserve instruction files); surface the mode and waiting state in the UI; document the `.multica` convention. The pure modules land before the daemon wiring so the daemon imports tested code.
- **Risk:** the gate-vs-locker interaction (worktree gate and in-place locker must not double-block) is the sharpest risk; the waiting-status integration test asserts parallel worktree tasks and serial in-place tasks explicitly. The Windows path-validation branch is the second; its table test must run on the owner's machine.
- **Provenance:** the path validator, locker, and waiting status are ported from PR #2's local-directory work (head `973a4392`); the workspace-level mode, the manifest `mode` field, the preserved-flow branch preparation, and the UI indicator are fork-original design.
