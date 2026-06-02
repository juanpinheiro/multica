import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";
import type { HandoffState } from "../types";

export const handoffKeys = {
  all: (issueId: string) => ["handoffs", issueId] as const,
  list: (issueId: string) => [...handoffKeys.all(issueId), "list"] as const,
};

export function handoffListOptions(issueId: string) {
  return queryOptions({
    queryKey: handoffKeys.list(issueId),
    queryFn: () => api.listHandoffs(issueId),
    select: (data) => data.handoffs,
    enabled: !!issueId,
  });
}

// latestState derives the current state from an ordered list of Handoffs.
// Mirrors server/internal/handoff.LatestState: the last Handoff's view is the
// authoritative current state (the agent accumulates history in each Handoff).
export function latestState(
  handoffs: { done: string[]; left_undone: string[]; discoveries: string[] }[]
): HandoffState {
  const last = handoffs[handoffs.length - 1];
  if (!last) {
    return { done: [], left_undone: [], discoveries: [] };
  }
  return {
    done: last.done ?? [],
    left_undone: last.left_undone ?? [],
    discoveries: last.discoveries ?? [],
  };
}
