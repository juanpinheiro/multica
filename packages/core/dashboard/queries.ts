import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";

export const dashboardKeys = {
  all: (wsId: string) => ["dashboard", wsId] as const,
  daily: (
    wsId: string,
    days: number,
    featureId: string | null,
    tz: string,
  ) => [...dashboardKeys.all(wsId), "daily", days, featureId, tz] as const,
  byAgent: (
    wsId: string,
    days: number,
    featureId: string | null,
    tz: string,
  ) => [...dashboardKeys.all(wsId), "by-agent", days, featureId, tz] as const,
  agentRuntime: (
    wsId: string,
    days: number,
    featureId: string | null,
    tz: string,
  ) => [...dashboardKeys.all(wsId), "agent-runtime", days, featureId, tz] as const,
  runTimeDaily: (
    wsId: string,
    days: number,
    featureId: string | null,
    tz: string,
  ) => [...dashboardKeys.all(wsId), "runtime-daily", days, featureId, tz] as const,
};

// 5-min rollup cadence on the server, 60s background refetch on the client.
const STALE_TIME = 60 * 1000;

// `tz` participates in every dashboard key so a Preferences change
// repoints the cache. All four series — token rollups and the
// atq.completed_at-based run-time series — slice their day boundary in
// the viewer's tz, so the four dashboard tabs always agree.
export function dashboardUsageDailyOptions(
  wsId: string,
  days: number,
  featureId: string | null,
  tz: string,
) {
  return queryOptions({
    queryKey: dashboardKeys.daily(wsId, days, featureId, tz),
    queryFn: () =>
      api.getDashboardUsageDaily({
        days,
        feature_id: featureId ?? undefined,
        tz,
      }),
    enabled: !!wsId,
    staleTime: STALE_TIME,
  });
}

export function dashboardUsageByAgentOptions(
  wsId: string,
  days: number,
  featureId: string | null,
  tz: string,
) {
  return queryOptions({
    queryKey: dashboardKeys.byAgent(wsId, days, featureId, tz),
    queryFn: () =>
      api.getDashboardUsageByAgent({
        days,
        feature_id: featureId ?? undefined,
        tz,
      }),
    enabled: !!wsId,
    staleTime: STALE_TIME,
  });
}

export function dashboardAgentRunTimeOptions(
  wsId: string,
  days: number,
  featureId: string | null,
  tz: string,
) {
  return queryOptions({
    queryKey: dashboardKeys.agentRuntime(wsId, days, featureId, tz),
    queryFn: () =>
      api.getDashboardAgentRunTime({
        days,
        feature_id: featureId ?? undefined,
        tz,
      }),
    enabled: !!wsId,
    staleTime: STALE_TIME,
  });
}

export function dashboardRunTimeDailyOptions(
  wsId: string,
  days: number,
  featureId: string | null,
  tz: string,
) {
  return queryOptions({
    queryKey: dashboardKeys.runTimeDaily(wsId, days, featureId, tz),
    queryFn: () =>
      api.getDashboardRunTimeDaily({
        days,
        feature_id: featureId ?? undefined,
        tz,
      }),
    enabled: !!wsId,
    staleTime: STALE_TIME,
  });
}
