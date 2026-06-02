# Entity model: Initiative → Milestone → Issue → Run

Work is modeled as **Initiative** (the PRD-level container — one shared branch, one PR)
→ **Milestone** (an ordered group of Issues that ends in a validation phase) → **Issue**
(a unit of work, sized to one context window, carrying a `Model` preference) → **Run**
(one execution of an Agent on an Issue, with a `role` of `worker` or `validator`).

A **Validator is a mode of a Run, not a separate entity** — the same executor invoked
with clean context (creator-verifier). One Issue accumulates many Runs over its life
(initial attempt, retry, validation, follow-up); each Run produces a Handoff.

We **reshape upstream tables** (`feature`→Initiative, `issue`→Issue,
`agent_task_queue`→Run + a `role` column) and **add new tables** (Milestone, Handoff,
DoD/assertions, Decision Log) rather than greenfielding the schema, because the daemon's
DB access is reused wholesale (see ADR-0001).

## Consequences

- The board shows **Issue status** (the durable "what"); the live monitor watches the
  **active Run** (the ephemeral "doing"). These are the two axes of the execution
  monitor — status × liveness.
- Names are ours (not upstream's `feature`, not Factory's `Mission`); see CONTEXT.md.
