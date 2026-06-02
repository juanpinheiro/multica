import type { AgentTask, IssueStatus } from "../types";

export type LivenessPhase = "claim" | "run" | "push" | "pr";

export interface Liveness {
  active: boolean;
  phase: LivenessPhase;
  heartbeat: "fresh" | "quiet";
  quietMs: number;
  elapsedMs: number;
  waiting: { reason: string; holderKey: string | null } | null;
}

export interface LivenessCtx {
  issueStatus?: IssueStatus;
  hasPr?: boolean;
  holderKey?: string | null;
}

// A running agent is considered "quiet" once this much time passes with no
// new task:message. Surfaces a possible stall without claiming the task has
// failed — long tool calls can legitimately exceed shorter windows.
export const QUIET_THRESHOLD_MS = 10_000;

function derivePhase(task: AgentTask, ctx?: LivenessCtx): LivenessPhase {
  if (task.status === "waiting_local_directory") return "claim";
  if (ctx?.hasPr || ctx?.issueStatus === "in_review") return "pr";
  if (task.status === "running") return "run";
  return "claim";
}

// Time since the agent's last observed activity. Falls back to started_at
// when no task:message has arrived yet, so a freshly-claimed run reads as
// fresh rather than stalled. Null on both → 0 (nothing to measure yet).
function deriveQuietMs(task: AgentTask, now: number): number {
  const baseline = task.last_activity_at ?? task.started_at;
  if (!baseline) return 0;
  return Math.max(0, now - new Date(baseline).getTime());
}

export function deriveLiveness(task: AgentTask, now: number, ctx?: LivenessCtx): Liveness {
  const active =
    task.status === "running" || task.status === "waiting_local_directory";

  const elapsedMs = task.started_at
    ? Math.max(0, now - new Date(task.started_at).getTime())
    : 0;

  const quietMs = deriveQuietMs(task, now);

  const waiting =
    task.status === "waiting_local_directory"
      ? {
          reason: task.wait_reason ?? "waiting for umbrella lock",
          holderKey: ctx?.holderKey ?? null,
        }
      : null;

  return {
    active,
    phase: derivePhase(task, ctx),
    heartbeat: quietMs > QUIET_THRESHOLD_MS ? "quiet" : "fresh",
    quietMs,
    elapsedMs,
    waiting,
  };
}
