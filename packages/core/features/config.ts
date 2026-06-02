import type { FeatureStatus, FeaturePriority } from "../types";

export const FEATURE_STATUS_ORDER: FeatureStatus[] = [
  "draft",
  "ready",
  "running",
  "in_review",
  "done",
  "blocked",
  "cancelled",
];

export const FEATURE_STATUS_CONFIG: Record<
  FeatureStatus,
  { label: string; color: string; dotColor: string; badgeBg: string; badgeText: string }
> = {
  draft: { label: "Draft", color: "text-muted-foreground", dotColor: "bg-muted-foreground", badgeBg: "bg-muted", badgeText: "text-muted-foreground" },
  ready: { label: "Ready", color: "text-info", dotColor: "bg-info", badgeBg: "bg-info", badgeText: "text-white" },
  running: { label: "Running", color: "text-warning", dotColor: "bg-warning", badgeBg: "bg-warning", badgeText: "text-white" },
  in_review: { label: "In Review", color: "text-info", dotColor: "bg-info", badgeBg: "bg-info", badgeText: "text-white" },
  done: { label: "Done", color: "text-info", dotColor: "bg-info", badgeBg: "bg-info", badgeText: "text-white" },
  blocked: { label: "Blocked", color: "text-muted-foreground", dotColor: "bg-muted-foreground", badgeBg: "bg-muted", badgeText: "text-muted-foreground" },
  cancelled: { label: "Cancelled", color: "text-destructive", dotColor: "bg-destructive", badgeBg: "bg-muted", badgeText: "text-muted-foreground" },
};

export const FEATURE_PRIORITY_ORDER: FeaturePriority[] = [
  "urgent",
  "high",
  "medium",
  "low",
  "none",
];

export const FEATURE_PRIORITY_CONFIG: Record<
  FeaturePriority,
  { label: string; bars: number; color: string; badgeBg: string; badgeText: string }
> = {
  urgent: { label: "Urgent", bars: 4, color: "text-destructive", badgeBg: "bg-destructive/10", badgeText: "text-destructive" },
  high: { label: "High", bars: 3, color: "text-warning", badgeBg: "bg-warning/10", badgeText: "text-warning" },
  medium: { label: "Medium", bars: 2, color: "text-warning", badgeBg: "bg-warning/10", badgeText: "text-warning" },
  low: { label: "Low", bars: 1, color: "text-info", badgeBg: "bg-info/10", badgeText: "text-info" },
  none: { label: "No priority", bars: 0, color: "text-muted-foreground", badgeBg: "bg-muted", badgeText: "text-muted-foreground" },
};
