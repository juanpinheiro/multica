"use client";

import { Check } from "lucide-react";
import {
  FEATURE_STATUS_CONFIG,
  FEATURE_STATUS_ORDER,
  FEATURE_PRIORITY_CONFIG,
  FEATURE_PRIORITY_ORDER
} from "@multica/core/features/config";
import { cn } from "@multica/ui/lib/utils";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@multica/ui/components/ui/dropdown-menu";
import type { Feature, FeatureStatus, FeaturePriority, UpdateFeatureRequest } from "@multica/core/types";
import { PriorityIcon } from "../../issues/components/priority-icon";
import { useFeatureStatusLabels, useFeaturePriorityLabels } from "./labels";

export function FeatureStatusBadge({ feature, handleUpdate, triggerClassName, align = "end" }: { feature: Feature; handleUpdate: (data: UpdateFeatureRequest) => void; triggerClassName?: string; align?: "start" | "end" | "center" }) {
  const statusLabels = useFeatureStatusLabels();
  const statusCfg = FEATURE_STATUS_CONFIG[feature.status];

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <button type="button" className={cn(
            "inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-xs font-medium cursor-pointer hover:opacity-80 transition-opacity",
            statusCfg.badgeBg, statusCfg.badgeText,
            triggerClassName
          )}>
            {statusLabels[feature.status]}
          </button>
        }
      />
      <DropdownMenuContent align={align} className="w-44">
        {FEATURE_STATUS_ORDER.map((s) => (
          <DropdownMenuItem key={s} onClick={() => handleUpdate({ status: s as FeatureStatus })}>
            <span className={cn("size-2 rounded-full", FEATURE_STATUS_CONFIG[s].dotColor)} />
            <span>{statusLabels[s]}</span>
            {s === feature.status && <Check className="ml-auto h-3.5 w-3.5" />}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function FeaturePriorityBadge({ feature, handleUpdate, triggerClassName, align = "end" }: { feature: Feature; handleUpdate: (data: UpdateFeatureRequest) => void; triggerClassName?: string; align?: "start" | "end" | "center" }) {
  const priorityLabels = useFeaturePriorityLabels();
  const priorityCfg = FEATURE_PRIORITY_CONFIG[feature.priority];

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <button type="button" className={cn(
            "inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-xs font-medium hover:bg-accent/60 transition-colors cursor-pointer",
            triggerClassName
          )}>
            <PriorityIcon priority={feature.priority} />
            <span className={cn("text-xs", priorityCfg.color)}>{priorityLabels[feature.priority]}</span>
          </button>
        }
      />
      <DropdownMenuContent align={align} className="w-44">
        {FEATURE_PRIORITY_ORDER.map((p) => (
          <DropdownMenuItem key={p} onClick={() => handleUpdate({ priority: p as FeaturePriority })}>
            <PriorityIcon priority={p} />
            <span>{priorityLabels[p]}</span>
            {p === feature.priority && <Check className="ml-auto h-3.5 w-3.5" />}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
