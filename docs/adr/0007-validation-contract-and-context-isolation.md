# Validation by a DoD checked by clean-context validators; context isolation by fresh Runs + Handoffs

**"Done" is defined by a Definition of Done (DoD)** — a coherent set of assertions
written at planning time, against the goal, **independent of any implementation**.
Acceptance Criteria is the per-Issue *view* of the DoD, not an independent,
implementation-shaped checklist (the latter confirms decisions instead of catching bugs).
Assertions are tagged to Milestones.

**Validator Runs** (clean context, a different Agent from the implementing worker) check
the accumulated work against the DoD at each Milestone boundary; the Orchestrator creates
follow-up Issues when assertions fail and gates progress until the DoD is green.

The **"dumbzone" (context degradation as a window fills) is avoided structurally**: each
Run starts with **fresh context** and reads the durable state (Handoffs + git + Issue
spec + DoD) rather than inheriting accumulated tokens. **Issues are sized to one context
window** (the analog of sizing an Initiative to one reviewable PR), and a **runtime
guard** makes overflow a non-event — a Run nearing its limit writes a Handoff and exits,
and a fresh continuation Run takes over.

## Consequences

- The DoD is the safety net of autonomous (AFK) runs (ADR-0005), so its quality at
  planning time carries the whole risk — which is why the planning conversation is the
  product's real leverage, not polish.
