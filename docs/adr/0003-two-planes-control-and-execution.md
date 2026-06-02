# Two planes: control (human + Claude via MCP) and execution (daemon + agents)

The system splits into two planes:

- **Control plane** — the human and Claude Code, acting through the **MCP**. They create
  Initiatives, write the PRD and DoD, decompose into Milestones/Issues, and set status.
  Human-in-the-loop lives entirely here.
- **Execution plane** — the daemon, Orchestrator, and Agents. They claim every `ready`
  Initiative and work it autonomously: dispatch, validation, PR. Autonomy lives entirely
  here.

The trigger is **status-driven**: agents pick up any Initiative the control plane has
flipped to `ready` — Ralph's `select_next` (today a `grep` for `ready-for-agent` over
markdown) lifted to the Initiative level and backed by the database. Initiative status:
`draft` → `ready` (the gate) → `running` → `in_review` (PR open, waiting on the human) →
`done`, plus `blocked` / `cancelled`.

## Consequences

- Initiative sequencing/dependencies are resolved by the control plane setting status
  (you mark B `ready` once A is merged), not by an in-system scheduler. Claude Code can
  automate this via the MCP, but control sits with the human by default.
