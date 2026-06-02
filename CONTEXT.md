# Initiative Runner (working name; forked from multica)

A system where a human plans work engineer-to-engineer (a PRD, issues, architecture guidelines) and
then autonomous agents execute it on a local daemon, producing a PR that validation agents have
already brought to its best state before the human's team reviews it.

It fuses three things, plus one borrowed idea:

- **`/to-prd` + `/to-issues`** — planning skills that write issues as markdown, each with a `Model`
  preference and a status.
- **`ralph/afk.sh`** — a serial loop (pick the next ready unit, run an agent, repeat). The proof that
  serial + model-per-issue is fast. A *pattern*, not the destination.
- **Factory Missions** — the borrowed idea: an orchestrator + validation phases + contracts +
  checkpoints, which is what makes a run safe for days. (We deliberately do **not** reuse the name
  "Mission" — it is their product.)

## The two planes

- **Control plane — the human + Claude Code (via MCP).** Creates Initiatives, writes the PRD/DoD,
  decomposes into Milestones/Issues, and sets status. Where human-in-the-loop lives. Flipping an
  Initiative to `ready` is the trigger. Org metaphor: CEO + chief of staff.
- **Execution plane — daemon + Orchestrator + Agents.** Claims every `ready` Initiative and works it
  autonomously: dispatch, validation, PR. Where autonomy lives. Org metaphor: COO + the team.

## The three layers

- **Platform (multica):** daemon, Postgres, Go server, MCP, web monitor, autopilot. The substrate.
  Stays. The DB is the source of truth.
- **Execution pattern (ralph):** serial loop + model-per-issue. Folds into the orchestrator's
  serial-with-internal-parallelism model.
- **Intelligence (missions):** orchestrator, validation, contracts, checkpoints, structured handoffs.
  The new layer grafted onto the platform.

## Language

**Initiative**:
The top-level container, defined by a PRD and decomposed into Milestones. The body of work that runs
autonomously for hours or days. **One Initiative = one PRD = several Milestones = one branch = one PR.**
Sized so its PR is reviewable in one sitting; a PRD too big for that splits into multiple Initiatives
at planning time (never into per-Milestone PRs). Maps to upstream's `feature` table, reshaped.
**Status** (set by the control plane): `draft` (being planned via MCP; agents ignore) → `ready` (the
gate — agents claim and start) → `running` (Orchestrator executing) → `in_review` (PR open, waiting on
the human) → `done` (PR merged); plus `blocked` / `cancelled`. Agents pick up every `ready` Initiative
and work it alone (Ralph's `select_next` lifted to the Initiative level).
_Avoid_: Mission (Factory's brand), Epic, Project, Feature.

**Milestone**:
A meaningful checkpoint within an Initiative — an ordered group of Issues that ends with a validation
phase. The boundary where the deterministic Gate blocks progress until validation passes.
_Avoid_: Sprint (implies a time-box; ours is scope-boxed), Phase.

**Issue**:
A unit of work, belonging to a Milestone. Carries a `Model` preference and Acceptance Criteria. The
durable "what" — what the human and Orchestrator plan and reason about. **Sized to fit one context
window** (just as an Initiative is sized to one reviewable PR): an Issue too big for a single fresh
Run is decomposed at planning. Heavy Issues can pick a larger-context `Model`.
_Avoid_: Story, Ticket, Task (a Task is an execution, see Run).

**Run**:
A single execution of an **Agent** on an Issue, with a `role` (worker | validator). Each Run starts
with **fresh context** — it never inherits the previous Run's accumulated tokens; it reads the durable
state (Handoffs + git + Issue spec + DoD) instead. This per-Run context reset is the primary defense
against the "dumbzone" (attention degradation as a window fills). One Issue accumulates many Runs
(initial attempt, retry, validation, follow-up); each Run produces a Handoff. A Run targets **exactly
one Repo** (its Issue's repo) and only ever writes there — cross-repo work is several Issues, not one Run.
The ephemeral "doing" — what the daemon executes and the live monitor watches.
_Avoid_: Task, Job, Attempt (an attempt is one kind of Run).

**Repo**:
A single git repository inside a Workspace's **Umbrella**, declared in `.multica/workspace.toml`. Each
**Issue** targets exactly one Repo; a **Run** only ever writes to its Issue's Repo. A cross-repo
Initiative spans several Repos via sibling Issues, coordinated by a shared `feature/<slug>` branch.
_Avoid_: project, module, package.

**Umbrella**:
The directory that holds `.multica/workspace.toml` and contains a Workspace's Repos as children. One
Workspace = one Umbrella. In `in_place` **Execution Mode** the Umbrella is the agent's working root, so a
Run can *read* sibling Repos for cross-repo alignment (it still only *writes* its own Repo).
_Avoid_: root, monorepo (the Repos stay independent git repositories).

**Execution Mode** (worktree | in_place):
A **Workspace-level** choice (declared once in the manifest, applied to all its Repos) for *where* Runs
execute. `worktree` = each Run in an isolated per-Repo clone — parallel across Runs, and the Umbrella is
read-only planning context. `in_place` = Runs in the real Umbrella on disk — serial across the whole
Umbrella, requires a clean tree, and the human watches it live. Unrelated to **Mode** (HITL|AFK), which is
a planning choice.
_Avoid_: per-repo mode (it is workspace-level); confusing with Mode (HITL|AFK).

**Agent**:
The actor — a configured persona/runtime (model, instructions, skills) that executes Runs. First-class
in multica already. An Agent runs an Issue in worker mode or validator mode.
_Avoid_: Worker, Developer, Bot.

**Orchestrator**:
An Agent (prompt/skill-driven, not a hardcoded state machine) that **conducts the execution of one
`ready` Initiative** — the COO of that delivery. It does NOT plan: the PRD/DoD/Milestone/Issue
decomposition is control-plane work (human + Claude Code via MCP). It dispatches Issues respecting
dependencies, triggers the validator at each Milestone boundary, and creates **reactive follow-up
Issues** when validation fails — gating progress until the DoD is green. Reasons in language; no fixed
transition table. **Stateless**: re-invoked at boundaries, reads the durable state (board + Handoffs)
fresh, never one long session — survives machine restarts because no state lives in its head. One
Orchestrator per in-flight Initiative; cross-Initiative sequencing is the control plane (status flips),
not the Orchestrator.
_Avoid_: coordinator, manager, squad leader (squad is being removed), planner (it executes, not plans).

**Gate** (deterministic bookkeeping):
The thin, non-LLM rule that blocks claiming the next Milestone's work until the current Milestone's
validation has passed. Inherited from multica's claim gate, extended to Milestone boundaries. The
*only* deterministic part of orchestration; everything else is the Agent reasoning.

**Definition of Done** (DoD):
The Initiative-level contract: a coherent set of assertions written during planning, against the
goal, independent of any implementation. The source of truth a validator Run checks against. Assertions
are tagged to Milestones (and optionally Issues).
_Avoid_: spec, requirements.

**Acceptance Criteria**:
The per-Issue view of "done." A derived recital of the DoD assertions that touch this Issue — not an
independent, implementation-shaped checklist.

**Handoff**:
The structured record an Agent writes when finishing a Run: what was completed, what was left undone,
which commands ran and their exit codes, issues discovered. The durable state the stateless Orchestrator
reads on wake. Upstream re-derives context from a snapshot instead; we make the handoff explicit.

**Decision Log**:
The self-evolving layer of architectural decisions (ADRs), managed separately from the code and kept
current by agents — the "retrospective" ritual: at the Initiative boundary a **retrospective Run**
(a third Run `role` alongside worker/validator) revisits technical decisions, records what was learned,
and updates the ADRs / CONTEXT terms. Persisted as the `decision_log` table; each entry links back to
the ADR numbers and CONTEXT glossary terms it touches.
_Avoid_: Architecture Memory (was the fuzzy working term), docs.

**Mode** (HITL | AFK):
A **planning-time decomposition choice** (made in the `/to-prd` + `/to-issues` flow) for how a PRD
becomes Initiatives. It does NOT add a runtime gate — every Initiative always runs autonomously to its
PR. The choice is purely how finely you carve:
- **AFK** (away-from-keyboard): the whole PRD becomes **one Initiative** → all Milestones run and
  validate autonomously → **one large PR** at the end (Ralph-style). The human reviews once. Max
  autonomy. The DoD is the only safety net, so AFK planning interrogates the DoD hard. Cron-triggered
  Initiatives are always AFK and require a recipe DoD.
- **HITL** (human-in-the-loop): the PRD is **split into several Initiatives** → several smaller PRs,
  reviewed and merged in sequence. The human is "in the loop" via the review+merge between Initiatives,
  which also gates dependent Initiatives (D can't start until A/B/C are merged). More control; merging a
  PR can auto-trigger the next dependent Initiative (autopilot-on-merge).
Both: Milestones are always *internal* agent checkpoints; the **PR is the only human gate** — HITL just
creates more of them by splitting. The per-Initiative tripwire/budget (pause-and-ping on repeated
validation failure or budget cap) applies to any autonomous run regardless of Mode.
_Avoid_: confusing with **Execution Mode** (worktree|in_place), a separate execution-substrate choice.

**Autopilot**:
A trigger (cron / webhook / manual) that fires an Initiative. Inherited from multica; a kept primitive.
Always fires AFK Initiatives (no human present), so it requires a pre-written recipe DoD.

**PR boundary** (resolved):
One **consolidated PR per Repo** per Initiative — the human↔agent boundary, where the team reviews. A
single-Repo Initiative is one PR (the common case); a cross-repo Initiative yields one consolidated PR
*per Repo* on the shared `feature/<slug>` branch (git PRs are repo-scoped — there is no atomic cross-repo
PR), and within each Repo every Issue of the Initiative converges into that one PR. Opened as a **draft early**
(team can watch/comment, not asked to review), flipped to **ready-for-review only when the whole DoD is
green** across all Milestones. Milestones are *internal* agent checkpoints (validation against the DoD,
orchestrator self-heals), never human PR reviews — so the run stays autonomous until the PR. Per-Milestone
PRs are rejected: they force either human merge-gates between Milestones (breaks autonomy) or stacked PRs.
Reviewability is controlled by Initiative size, not by splitting the PR at Milestones.

## Open design questions (not glossary — for the build)

- **Branch & parallelism (RESOLVED).** Serial Runs *within* an Initiative (keep the claim Gate;
  matches Factory's hard-won lesson). Real parallelism is **across** Initiatives (each its own
  worktree/branch — multica already does this). **Within** a Run, acceleration comes from the agent
  CLI's own read-only sub-agents (Claude Code's Task/Explore agents) — we don't build it; we may inject
  custom read-only sub-agent definitions per Run via execenv. Invariant: sub-agents are read-only
  (explore/verify); only the main Run agent writes. The validator Run fans out one sub-agent per DoD
  assertion. Caveat: sub-agent capability is provider-dependent, so `Model` choice also picks fan-out
  power (Claude Code strong; Codex/Gemini vary — verify before committing).
- **Entity implementation.** Reshape upstream tables (`feature`→Initiative, `issue`→Issue,
  `agent_task_queue`→Run+`role`) + new tables (Milestone, Handoff, DoD/assertions, Decision Log).
  Direction agreed; details pending.
- **Dead on arrival.** Squad (being removed), `member` assignee type, login/user-identity chrome,
  manual "New Issue" CTA. Confirmed cuts.
- **Human↔agent boundary (RESOLVED).** A planning-time **Mode** (HITL | AFK) = how finely the PRD is
  carved into Initiatives. AFK = one Initiative → one big autonomous PR (Ralph-style). HITL = many
  Initiatives → many PRs reviewed/merged in sequence. Milestones stay internal; the PR is the only human
  gate. See **Mode**.
- **Cron-triggered planning (RESOLVED).** Cron fires AFK Initiatives only, which require a pre-written
  recipe DoD — so the missing planning conversation is replaced by a reusable DoD template.

## Example dialogue

> **Human:** New Initiative — add rate-limiting across the API. Here's the PRD.
> **Orchestrator:** I'll decompose it into three Milestones: middleware, config, docs. The DoD has an
> assertion "requests over the limit get a 429 with Retry-After" — I've tagged it to the middleware
> Milestone. Middleware and config are independent, so their Issues run as parallel Runs in separate
> worktrees; docs is one Issue gated behind both.
> **Human:** Go.
> *(later)*
> **Orchestrator (woken at the middleware Milestone boundary):** A validator Run checked the accumulated
> work against the DoD. The 429 path works but it's missing Retry-After. I'm creating a follow-up Issue;
> the config Milestone stays gated until it passes. Nothing reaches your team's PR until the DoD is green.
