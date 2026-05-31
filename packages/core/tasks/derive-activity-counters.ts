export interface TaskMessage {
  type: "text" | "thinking" | "tool_use" | "tool_result" | "error";
}

export interface ActivityCounters {
  activityCount: number;
  elapsedMs: number;
}

export function deriveActivityCounters(
  messages: readonly TaskMessage[],
  startedAt: string | null,
  now: number,
): ActivityCounters {
  let activityCount = 0;
  for (const msg of messages) {
    if (msg.type === "tool_use") activityCount++;
  }

  const elapsedMs = startedAt
    ? Math.max(0, now - new Date(startedAt).getTime())
    : 0;

  return { activityCount, elapsedMs };
}
