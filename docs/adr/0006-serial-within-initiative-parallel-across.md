# Serial Runs within an Initiative; parallelism across Initiatives + read-only sub-agent fan-out

Within an Initiative (one shared branch), **Runs are serial**. The claim Gate already
enforces this, and Factory's experience is explicit: parallel writers on one branch trade
correctness for throughput ("agents conflict, step on each other's changes; coordination
overhead eats the speed gains").

**Real parallelism is across Initiatives**, each in its own worktree/branch — which
multica already supports (the Gate only serializes same-branch Runs).

**Acceleration within a single Run** comes from the agent CLI's own **read-only
sub-agents** (Claude Code's Task/Explore agents), which the daemon already invokes and
which we may define per Run via execenv. The invariant: sub-agents are **read-only**
(explore/verify); only the main Run agent writes — otherwise the same-branch collision
returns. The validator Run fans out one sub-agent per DoD assertion.

## Consequences

- Sub-agent capability is **provider-dependent**, so an Issue's `Model` choice also picks
  its fan-out power (Claude Code strong; Codex/Gemini vary — verify before relying on it).
- Sub-agents also serve context hygiene: they absorb token-heavy reads and return
  summaries, keeping the main Run lean (see ADR-0007).
