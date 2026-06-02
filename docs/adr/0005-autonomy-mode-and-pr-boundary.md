# Autonomy is per-Initiative (Mode HITL/AFK); one PR per Initiative is the only human gate

Instead of a global autonomy policy, each PRD is **carved into Initiatives at planning
time**, and that carving *is* the autonomy choice:

- **AFK** — the whole PRD becomes **one Initiative** → all Milestones run and validate
  autonomously → **one large PR** at the end (Ralph-style), reviewed once. Max autonomy;
  the DoD is the only safety net, so AFK planning interrogates the DoD hard.
- **HITL** — the PRD is **split into several Initiatives** → several smaller PRs,
  reviewed and merged in sequence; the merges gate dependent Initiatives. More control.

**Milestones are always internal agent checkpoints** (validation against the DoD); the
**PR is the only human review gate** — HITL simply creates more of them by splitting.
Reviewability is controlled by **Initiative size**, never by per-Milestone PRs.

A per-Initiative **tripwire** (repeated validation failure on the same Milestone, or a
budget cap on tokens/Runs/time) **pauses-and-pings** the human rather than burning days,
making AFK safe even with an imperfect DoD. Cron/autopilot triggers are always AFK and
require a pre-written recipe DoD.

## Considered Options

- **Per-Milestone human checkpoints inside one Initiative** — rejected. Adds a second
  gate type; the same control is achieved by splitting into more Initiatives (HITL),
  reusing the one boundary that already exists (the PR).
- **Per-Milestone PRs** — rejected. Forces either human merge-gates between Milestones
  (breaks autonomy) or stacked PRs (complex to review and merge).
