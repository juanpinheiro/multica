"use client";

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { agentTaskSnapshotOptions } from "../agents/queries";
import { inboxListOptions } from "../inbox/queries";
import { issueListOptions } from "../issues/queries";
import type { AgentTask, InboxItem, Issue } from "../types";
import { buildLiveEvents, type BuildLiveEventsResult } from "./build-live-events";

const EMPTY_TASKS: AgentTask[] = [];
const EMPTY_INBOX: InboxItem[] = [];
const EMPTY_ISSUES: Issue[] = [];

// Aggregates the workspace's task snapshot, inbox, and issue list into a
// single newest-first activity feed. Pure derivation off TanStack caches —
// no extra network. WS-driven cache invalidation keeps events fresh.
export function useLiveEvents(wsId: string, now: number = Date.now()): BuildLiveEventsResult {
  const { data: tasks = EMPTY_TASKS } = useQuery({
    ...agentTaskSnapshotOptions(wsId),
    enabled: !!wsId,
  });
  const { data: inbox = EMPTY_INBOX } = useQuery({
    ...inboxListOptions(wsId),
    enabled: !!wsId,
  });
  const { data: issues = EMPTY_ISSUES } = useQuery({
    ...issueListOptions(wsId),
    enabled: !!wsId,
  });
  return useMemo(
    () => buildLiveEvents({ tasks, inbox, issues, now }),
    [tasks, inbox, issues, now],
  );
}
