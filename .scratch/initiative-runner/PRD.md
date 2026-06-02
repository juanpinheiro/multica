# PRD: Initiative Runner

**Status:** `ready-for-agent`
**Owner:** Juan Pinheiro
**Created:** 2026-05-31

> Glossary: `CONTEXT.md`. Architecture decisions: `docs/adr/0001`–`0007`. This PRD is itself a
> **HITL, multi-Initiative effort** — `/to-issues` should carve it into several Initiatives (suggested
> carve in Further Notes), not one giant PR.

## Problem Statement

Today the author drives autonomous agent work two ways, and neither is enough on its own:

- **The multica fork** is a robust platform — local daemon that runs agent CLIs (Claude Code, Codex,
  Gemini, …), a Postgres-backed task queue with a claim gate, worktree isolation, crash recovery, an
  MCP server, and a real-time web monitor. But it is shaped as a multi-tenant SaaS for human teams, and
  it has *no orchestration intelligence*: it dispatches one agent to one issue and that is the end of it.
- **`ralph/afk.sh`** is the author's real daily driver: a serial loop that reads markdown issues from
  `.scratch/<feature>/`, picks the next `ready-for-agent` one, runs `claude --print --model <model>`, and
  repeats. It is fast and already does model-per-issue — but it is dumb: one issue at a time, no
  orchestrator, no validation, no notion of "done", no safety to run for days.

The author wants to **plan work engineer-to-engineer (a PRD, the architecture, the guidelines), approve
it, and then have agents run alone for hours or days — producing a PR that validation agents have already
brought to its best state before a human team reviews it.** Neither tool does this. The missing piece is
the *intelligence layer* from Factory Missions (an orchestrator + validation contracts + milestone
checkpoints) — the thing that makes a multi-day run safe instead of drifting into garbage.

## Solution

Fuse the three — the multica platform (substrate), the Ralph pattern (serial execution + model-per-issue),
and Factory Missions (the intelligence) — into an **Initiative Runner**, built by reshaping the existing
fork (ADR-0001), not greenfield.

Work is modeled as **Initiative → Milestone → Issue → Run** (ADR-0002). The system splits into **two
planes** (ADR-0003):

- **Control plane (the human + Claude Code, via the MCP):** create an Initiative, write its PRD and its
  **Definition of Done**, decompose it into Milestones and Issues, and set status. Flipping an Initiative
  to `ready` is the trigger. This is where human-in-the-loop lives.
- **Execution plane (daemon + Orchestrator + Agents):** the daemon claims every `ready` Initiative; the
  **Orchestrator** (a stateless prompt-driven agent — the "COO" of one Initiative, ADR-0004) dispatches
  Issues, triggers a **validator Run** at each Milestone boundary to check the DoD, creates reactive
  follow-up Issues when validation fails, and opens one PR per Initiative when the whole DoD is green.

Autonomy is a **planning-time choice per PRD** (ADR-0005): **AFK** = one Initiative → one big autonomous
PR (Ralph-style, reviewed once); **HITL** = several Initiatives → several smaller PRs reviewed and merged
in sequence. Milestones are always *internal* agent checkpoints; the PR is the only human review gate.

Execution is **serial within an Initiative, parallel across Initiatives**, with read-only sub-agent
fan-out accelerating individual Runs (ADR-0006). Context degradation ("dumbzone") is avoided structurally:
each Run starts fresh and reads durable state (Handoffs + git + DoD); Issues are sized to one context
window (ADR-0007).

The fork is simultaneously **stripped** of everything the single-user, agent-driven, MCP-controlled model
makes dead: the multi-tenant/login chrome, squads, the `member` assignee, manual creation UI, the
human-collaboration social layer (reactions, subscribers, human @mentions), and ad-hoc chat.

## User Stories

**Control plane — planning & steering (human + Claude Code via MCP)**

1. As the author, I want to discuss a PRD with Claude Code as an engineer, so that the plan reflects real
   architectural judgment before any code is written.
2. As the author, I want Claude Code to create an Initiative via the MCP, so that I never have to fill out
   a creation form in the UI.
3. As the author, I want to write a Definition of Done at planning time — a set of assertions against the
   goal, independent of implementation — so that "done" is defined before the work starts.
4. As the author, I want each Issue to carry a `Model` preference, so that heavy Issues use a
   larger/stronger model and light Issues use a cheap one.
5. As the author, I want to decompose an Initiative into ordered Milestones, so that validation happens at
   meaningful checkpoints instead of only at the end.
6. As the author, I want to choose AFK (one big autonomous PR) or HITL (split into several reviewed PRs)
   when I plan, so that I match autonomy to the risk of the work.
7. As the author, I want to flip an Initiative to `ready`, so that agents pick it up and start — and so
   that NOT flipping it is how I hold work back.
8. As the author, I want dependent Initiatives to wait until I review and merge the ones they depend on,
   so that staged (HITL) efforts stay correct.
9. As the author, I want Claude Code to optionally flip the next Initiative to `ready` after a merge (via
   the MCP), so that a HITL chain can flow without me babysitting it — while control still defaults to me.
10. As the author, I want to define a budget (tokens/Runs/time) and failure tolerance per AFK Initiative,
    so that an imperfect DoD cannot burn days.

**Execution plane — autonomous run (daemon + Orchestrator + Agents)**

11. As the author, I want agents to claim every `ready` Initiative and start working alone, so that
    approved work runs without further input.
12. As the Orchestrator, I want to dispatch Issues respecting their dependencies, so that work happens in a
    safe order.
13. As the Orchestrator, I want to trigger a validator Run at each Milestone boundary, so that accumulated
    work is checked against the DoD before the next Milestone starts.
14. As the Orchestrator, I want to create follow-up Issues when validation surfaces problems, so that the
    Initiative self-heals instead of shipping a failing DoD.
15. As the Orchestrator, I want to be stateless — re-invoked at boundaries, reading durable state fresh —
    so that a machine restart mid-run loses nothing.
16. As a worker Agent, I want to read the Issue spec, the relevant Handoffs, and the DoD with fresh
    context, so that I never inherit another Run's accumulated tokens (no dumbzone).
17. As a worker Agent, I want to write a structured Handoff when I finish (done / left-undone / commands +
    exit codes / discoveries), so that the next Run and the Orchestrator know exactly where things stand.
18. As a validator Agent, I want clean context separate from the implementing worker, so that I review
    adversarially rather than confirming my own decisions.
19. As a validator Agent, I want to fan out one read-only sub-agent per DoD assertion, so that validation
    of a Milestone is fast and parallel.
20. As the author, I want Runs within an Initiative to be serial on a shared branch, so that agents never
    collide on the same code.
21. As the author, I want different Initiatives to run in parallel in separate worktrees, so that
    independent work proceeds concurrently.
22. As a Run, I want to checkpoint via a Handoff and exit before my context window fills, so that an
    oversized Issue degrades gracefully into a fresh continuation Run instead of getting dumb.
23. As the author, I want one PR per Initiative, opened as a draft early and flipped to ready-for-review
    only when the whole DoD is green, so that I (and my team) review a coherent, validated change once.
24. As the author, I want the system to pause-and-ping me when an Initiative trips its tripwire (repeated
    validation failure or budget cap), so that runaway runs stop and ask for help.
25. As the author, I want cron/autopilot triggers to fire AFK Initiatives from a pre-written recipe DoD, so
    that recurring work (daily audits, dependency bumps) runs safely without a planning conversation.

**Observe & review (the Mission Monitor UI)**

26. As the author, I want the home screen to be a live board where cards move as agents claim/work/finish,
    so that execution is visible instead of a black box.
27. As the author, I want each Initiative's view to show its PRD, Milestones, DoD, and status in one place,
    so that I can see the whole plan and its progress.
28. As the author, I want to see which DoD assertions passed or failed at each Milestone, so that I
    understand why the Orchestrator created a follow-up.
29. As the author, I want a per-Issue Run/Handoff timeline, so that I can read what each Run did, attempted,
    and discovered.
30. As the author, I want an inbox of PRs awaiting my review plus tripwire pauses, so that I catch up on
    what needs me without checking every page.
31. As the author, I want honest progress signals — a discrete phase stepper, a heartbeat that goes
    "quiet" when stalled, monotonic counters — instead of a fake percentage bar, so that I trust what I
    see.
32. As the author, I want to flip Initiative status from the UI too (a mirror of the MCP), so that I can
    steer without dropping into Claude Code.
33. As the author, I want a Mode (HITL/AFK) indicator and a Decision Log view, so that I can see how a run
    is governed and how technical decisions evolved.

**Strip the dead weight**

34. As the author, I do NOT want squads, the `member` assignee type, login/user-identity chrome, or a
    workspace-switcher-as-tenant-manager, so that the single-user reality is reflected.
35. As the author, I do NOT want manual issue/feature creation forms or human assignee pickers, since
    creation is the MCP's job.
36. As the author, I do NOT want reactions, subscribers, human @mentions, or ad-hoc chat sessions, since
    there is no second human and every agent invocation goes through the Initiative pipeline.
37. As the author, I do NOT want the onboarding wizard, email delivery, or API-token UI, since none apply
    to a single-user, locally-run tool.
38. As the author, I want agent @mentions in comments preserved as a dispatch trigger and comments
    preserved as how agents report progress, so that the agent-facing parts of the issue model stay.

## Implementation Decisions

### Entity model & schema (ADR-0002)

- **Reshape** upstream tables: `feature` → **Initiative** (+ status `draft|ready|running|in_review|done|
  blocked|cancelled`, + Mode `hitl|afk`, + budget/tripwire fields); `issue` → **Issue** (+ `milestone_id`,
  keep `Model` preference and `acceptance_criteria` as the per-Issue DoD view); `agent_task_queue` → **Run**
  (+ `role` `worker|validator`). Reuse the existing liveness columns (`last_activity_at`, `attempt`,
  `session_id`, `work_dir`, `failure_reason`).
- **New tables:** **Milestone** (ordered within an Initiative, with a validation status); **Handoff** (the
  structured Run output); **DoD assertion** (initiative-level, tagged to a Milestone); **Decision Log**
  (the self-evolving ADR layer, agent-maintained — see Further Notes).
- **Cut** the `chat_session_id` / quick-create alternate-parent paths on Run (chat is removed); a Run is
  always parented by an Issue.

### Deep modules (pure, isolatable — see Testing Decisions)

- **Gate** — `claimable(run, worldState) → ok | reason`. Encapsulates: Issue dependency gating, branch
  serialization (serial-within-Initiative), **Milestone gating** (don't open Milestone N+1 until N's
  validation passed), and the in-place umbrella lock. Extends multica's existing `ClaimAgentTask`
  predicate. The single deterministic core of orchestration (ADR-0004).
- **Tripwire/Budget** — `shouldPause(initiativeState) → pause | reason`. Pure accounting over
  per-Milestone failure counts and a token/Run/time cap. The AFK safety net (ADR-0005).
- **DoD evaluation** — `milestoneSatisfied(assertions, validatorResults) → bool` and the Initiative-level
  roll-up. Pure; defines what "done" means (ADR-0007).
- **Status state machine** — `transition(entity, from, to) → ok | err` for Initiative and Run. Pure;
  rejects illegal transitions.
- **Handoff store** — serialize/parse a Handoff and derive `latestState(issueId)` — the durable state the
  stateless Orchestrator reads on wake (ADR-0004).

### Orchestrator & Agents (to build)

- **Orchestrator** = a prompt/skill, not a Go state machine (ADR-0004). A thin server-side scaffold wakes
  it (subscribing to the existing `task:completed` event bus) and applies its decisions (dispatch /
  follow-up / gate) through the existing task-dispatch service. One Orchestrator Run per in-flight
  Initiative. It conducts; it does not decompose (decomposition is control-plane).
- **Validator role** = a Run dispatched with `role=validator` and clean context, prompted to check the
  Milestone's accumulated work against the DoD, fanning out one read-only sub-agent per assertion
  (ADR-0006). The creator-verifier separation is enforced by using a different Agent/context than the
  implementing worker.
- **Sub-agent fan-out** is delegated to the agent CLI (Claude Code's Task/Explore); read-only invariant —
  only the main Run writes. Custom read-only sub-agents may be injected per Run via execenv. Fan-out power
  is provider-dependent, so an Issue's `Model` choice also selects it.

### Control plane / MCP surface (extend existing MCP)

- MCP tools to: create/update Initiative, Milestone, Issue; write DoD assertions; set Initiative status
  (the `ready` flip and the rest of the state machine). The `/to-prd` and `/to-issues` skills are adapted
  to emit Initiatives/Milestones/Issues carrying `Model`, the DoD, and the Mode.
- Trigger is status-driven: the execution plane claims any `ready` Initiative (Ralph's `select_next`
  lifted to the Initiative level, DB-backed).

### Mission Monitor UI (build/reshape — this is a first-class module, not an afterthought)

- **Build/reshape:** the live board (extends `web-execution-monitor`), the Initiative view (PRD +
  Milestones + DoD + status), Milestone/DoD-assertion pass-fail display, the Run/Handoff timeline, the
  PR-review inbox (repurposed from the mentions feed), the status-flip control (UI mirror of the MCP), the
  Mode indicator + tripwire alerts, and the Decision Log view. Keep the honest-progress signals (phase
  stepper, heartbeat, counters — no fake % bar).
- **Cut (UI):** manual creation forms, human assignee pickers, `my-issues`, settings tabs for removed
  features, the chat sidebar, the tenant/login chrome, onboarding.

### Deadcode plan (ADR-0001)

- **Delete:** squad (handlers, schema, UI, `assignee_type=squad`); `member` assignee type and human
  assignee logic; login/signup/auth-identity chrome and `views/auth/`; manual creation UI; reactions;
  subscribers; human @mentions (keep agent-mention-as-dispatch-trigger); chat sessions + quick-create;
  onboarding; email delivery; API-token UI.
- This **subsumes and extends** the conservative `.scratch/multica-personal-fork` PRD (which stripped the
  multi-tenant/SaaS surface but kept all product features). Where they overlap (multi-tenant chrome,
  squads-adjacent), this PRD is the more aggressive authority; the personal-fork issues already marked
  `done` stand.

## Testing Decisions

- **TDD, red-green-refactor, on every deep module.** Write the failing test first, implement to green,
  refactor. The deep modules are pure or near-pure, so this is cheap and high-value.
- **Modules under test (all of them, per the author):** **Gate**, **Tripwire/Budget**, **DoD evaluation**,
  **Status state machine**, and the **Handoff store**.
- **What makes a good test here:** assert external behavior, not internals. For the Gate: given a world
  state, is this Run claimable, and if not, which reason? For Tripwire: given an Initiative's failure/budget
  state, pause or not? For DoD eval: given assertions + validator results, is the Milestone satisfied? For
  the Status SM: is this transition legal? For Handoffs: round-trip a record and derive the latest state.
  No reaching into private helpers.
- **Prior art:** Go handler tests in `server/internal/handler/*_test.go` (spin up a test DB, fire a
  request, assert on response/DB state) for anything DB-backed; Vitest + `@testing-library/react` in
  `packages/views/*.test.tsx` for UI; pure functions get plain table-driven `go test` / Vitest with no DB.
- **Safety net for the deletions:** the existing test suites for *kept* features are the regression net —
  a deletion that breaks a kept feature surfaces as a typecheck or test failure (the same strategy the
  personal-fork effort used).

## Out of Scope

- **Rename / rebrand.** The "multica" name is kept in code (binary, env vars, packages, module path, DB)
  for now; the Apache attribution must be preserved regardless (ADR-0001). A product rename is a separate
  later effort.
- **Public-internet hardening.** Local/trusted-network use only; no HTTPS termination, CSRF, etc.
- **Switching the platform.** Postgres stays; Next.js stays; no SQLite/embedded DB, no bundler swap.
- **New agent providers or new product features** beyond the Initiative Runner architecture.
- **Running the first real Initiative.** This PRD defines and builds the system; using it to drive work is
  the payoff, not part of this scope.

## Further Notes

- **Suggested HITL carve (`/to-issues` input):** (A) Schema + entity reshape + deadcode removal; (B) Gate +
  Status SM + Handoff store (the deterministic core); (C) Orchestrator scaffold + Validator role +
  sub-agent fan-out; (D) DoD + Tripwire/Budget + Mode; (E) MCP control-plane surface; (F) Mission Monitor
  UI. Dependencies roughly A → B → {C, D} → E → F, so this is naturally a staged (HITL) effort with a
  reviewed PR per Initiative.
- **The bitter-lesson stance (ADR-0004):** keep deterministic logic thin (Gate, recovery, dispatch
  bookkeeping) and put intelligence in prompts/skills, so the system improves with each model release.
- **Decision Log = the self-evolving "Architecture Memory":** an agent runs a retrospective at
  Milestone/Initiative boundaries, updating `docs/adr/` and `CONTEXT.md` and recording what was learned.
  Name is tentative; modeled as a new table + the existing `docs/adr/` files.
- **Provider dependence:** sub-agent fan-out (and therefore in-Run acceleration) is strong on Claude Code,
  varies on Codex/Gemini — verify per provider before relying on it for a given Issue's `Model`.

## Comments
