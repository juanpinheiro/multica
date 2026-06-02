# The Orchestrator is a stateless prompt-driven agent (the COO), not a state machine

Orchestration intelligence lives in **prompts/skills, not a hardcoded Go state
machine**, so the system improves with each model release (the "bitter lesson").

The Orchestrator is the **COO of one in-flight Initiative**. It *conducts execution* —
dispatches Issues respecting dependencies, triggers the validator at each Milestone
boundary, and creates **reactive follow-up Issues** when validation fails — but it does
**not plan**: the PRD/DoD/Milestone/Issue decomposition is control-plane work (ADR-0003).
There is one Orchestrator per in-flight Initiative; cross-Initiative sequencing is the
control plane, not the Orchestrator.

It is **stateless**: re-invoked at boundaries, it reads the durable state (the board +
Handoffs) fresh and never holds a long session, so it survives process and machine
restarts. The only deterministic code is the thin **Gate** — don't open the next
Milestone until the current one's validation passes — inherited from multica's claim
gate, extended to Milestone boundaries.

## Consequences

- The **Handoff** (an explicit, structured Run output: done / left-undone / commands +
  exit codes / discoveries) is load-bearing — it is the durable state the stateless
  Orchestrator reads on wake. Upstream re-derived context from a snapshot; we make it
  explicit.
