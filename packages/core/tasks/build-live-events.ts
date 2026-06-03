import type { AgentTask, InboxItem, Issue } from "../types";
import { deriveLiveness, type LivenessPhase } from "./derive-liveness";

export type ActivityEventType =
  | "agent_started"
  | "tool_use"
  | "edit"
  | "commit"
  | "milestone_passed"
  | "milestone_failed"
  | "issue_done"
  | "initiative_ready_for_review"
  | "dod_failed"
  | "tripwire_paused";

export interface ActivityEvent {
  id: string;
  at: string;
  type: ActivityEventType;
  initiativeId: string | null;
  issueId: string | null;
  agentId: string | null;
  message: string;
  phase?: LivenessPhase;
  heartbeat?: "fresh" | "quiet";
}

export interface BuildLiveEventsInput {
  tasks: readonly AgentTask[];
  inbox: readonly InboxItem[];
  issues: readonly Issue[];
  now: number;
}

export interface BuildLiveEventsResult {
  events: ActivityEvent[];
  runningAgents: number;
  runningInitiatives: number;
}

// Inbox -> activity type. Only the system-event slice the live feed cares
// about; chatty per-issue notifications (assignment changes, comments) stay
// out of the timeline.
const INBOX_TYPE_MAP: Partial<Record<InboxItem["type"], ActivityEventType>> = {
  task_completed: "issue_done",
  task_failed: "dod_failed",
  initiative_tripwire: "tripwire_paused",
  feature_ready_for_review: "initiative_ready_for_review",
};

function isRunning(task: AgentTask): boolean {
  return task.status === "running" || task.status === "waiting_local_directory";
}

function inboxFeatureId(
  item: InboxItem,
  issueIndex: Map<string, Issue>,
): string | null {
  const fromDetails = item.details?.feature_id;
  if (typeof fromDetails === "string" && fromDetails) return fromDetails;
  if (item.issue_id) return issueIndex.get(item.issue_id)?.feature_id ?? null;
  return null;
}

function featureIdOf(task: AgentTask, issueIndex: Map<string, Issue>): string | null {
  if (!task.issue_id) return null;
  return issueIndex.get(task.issue_id)?.feature_id ?? null;
}

function startedEvent(task: AgentTask, featureId: string | null): ActivityEvent | null {
  if (!task.started_at) return null;
  return {
    id: `task-started:${task.id}`,
    at: task.started_at,
    type: "agent_started",
    initiativeId: featureId,
    issueId: task.issue_id || null,
    agentId: task.agent_id,
    message: "",
  };
}

function activityEvent(
  task: AgentTask,
  featureId: string | null,
  now: number,
): ActivityEvent | null {
  if (!task.last_activity_at || task.last_activity_at === task.started_at) return null;
  const liveness = deriveLiveness(task, now);
  return {
    id: `task-activity:${task.id}`,
    at: task.last_activity_at,
    type: "tool_use",
    initiativeId: featureId,
    issueId: task.issue_id || null,
    agentId: task.agent_id,
    message: "",
    phase: liveness.phase,
    heartbeat: liveness.heartbeat,
  };
}

function buildRunningTaskEvents(
  tasks: readonly AgentTask[],
  issueIndex: Map<string, Issue>,
  now: number,
): ActivityEvent[] {
  const out: ActivityEvent[] = [];
  for (const task of tasks) {
    if (!isRunning(task)) continue;
    const featureId = featureIdOf(task, issueIndex);
    const started = startedEvent(task, featureId);
    if (started) out.push(started);
    const activity = activityEvent(task, featureId, now);
    if (activity) out.push(activity);
  }
  return out;
}

function buildInboxEvents(
  inbox: readonly InboxItem[],
  issueIndex: Map<string, Issue>,
): ActivityEvent[] {
  const out: ActivityEvent[] = [];
  for (const item of inbox) {
    const type = INBOX_TYPE_MAP[item.type];
    if (!type) continue;
    out.push({
      id: `inbox:${item.id}`,
      at: item.created_at,
      type,
      initiativeId: inboxFeatureId(item, issueIndex),
      issueId: item.issue_id ?? null,
      agentId: item.actor_type === "agent" ? item.actor_id : null,
      message: item.body || item.title,
    });
  }
  return out;
}

function dedupeAndSort(events: ActivityEvent[]): ActivityEvent[] {
  const seen = new Map<string, ActivityEvent>();
  for (const e of events) if (!seen.has(e.id)) seen.set(e.id, e);
  return Array.from(seen.values()).sort(
    (a, b) => new Date(b.at).getTime() - new Date(a.at).getTime(),
  );
}

function countRunningInitiatives(
  tasks: readonly AgentTask[],
  issueIndex: Map<string, Issue>,
): number {
  const featureIds = new Set<string>();
  for (const task of tasks) {
    if (!isRunning(task) || !task.issue_id) continue;
    const featureId = issueIndex.get(task.issue_id)?.feature_id;
    if (featureId) featureIds.add(featureId);
  }
  return featureIds.size;
}

export function buildLiveEvents(input: BuildLiveEventsInput): BuildLiveEventsResult {
  const issueIndex = new Map(input.issues.map((i) => [i.id, i]));
  const events = dedupeAndSort([
    ...buildRunningTaskEvents(input.tasks, issueIndex, input.now),
    ...buildInboxEvents(input.inbox, issueIndex),
  ]);
  return {
    events,
    runningAgents: input.tasks.filter(isRunning).length,
    runningInitiatives: countRunningInitiatives(input.tasks, issueIndex),
  };
}
