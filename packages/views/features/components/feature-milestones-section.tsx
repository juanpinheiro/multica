"use client";

import { useQuery } from "@tanstack/react-query";
import { CheckCircle2, Circle, XCircle } from "lucide-react";
import type { DodAssertion, Milestone } from "@multica/core/types";
import { featureMilestonesOptions } from "@multica/core/milestones/queries";
import { milestoneDodOptions } from "@multica/core/dod/queries";
import { useWorkspaceId } from "@multica/core/hooks";
import { cn } from "@multica/ui/lib/utils";
import { Skeleton } from "@multica/ui/components/ui/skeleton";
import { useT } from "../../i18n";

type ValidationStatus = Milestone["validation_status"];
type AssertionStatus = DodAssertion["status"];

function ValidationIcon({ status }: { status: ValidationStatus }) {
  if (status === "passed") return <CheckCircle2 className="h-3.5 w-3.5 text-success shrink-0" />;
  if (status === "failed") return <XCircle className="h-3.5 w-3.5 text-destructive shrink-0" />;
  return <Circle className="h-3.5 w-3.5 text-muted-foreground/40 shrink-0" />;
}

function assertionTextClass(status: AssertionStatus): string {
  if (status === "passed") return "text-success";
  if (status === "failed") return "text-destructive";
  return "text-muted-foreground/60";
}

function assertionMarker(status: AssertionStatus): string {
  if (status === "passed") return "✓";
  if (status === "failed") return "✗";
  return "○";
}

function MilestoneDoD({ milestoneId }: { milestoneId: string }) {
  const wsId = useWorkspaceId();
  const { data: assertions = [] } = useQuery(milestoneDodOptions(wsId, milestoneId));

  if (assertions.length === 0) return null;

  return (
    <div className="mt-1.5 space-y-0.5 pl-5">
      {assertions.map((a) => (
        <div key={a.id} data-testid={`dod-assertion-${a.id}`} className="flex items-start gap-1.5">
          <span className={cn("shrink-0 text-[10px] leading-4", assertionTextClass(a.status))}>
            {assertionMarker(a.status)}
          </span>
          <span className={cn("text-xs leading-snug", assertionTextClass(a.status))}>
            {a.text}
          </span>
        </div>
      ))}
    </div>
  );
}

function MilestoneRow({ milestone }: { milestone: Milestone }) {
  return (
    <div data-testid={`milestone-row-${milestone.id}`}>
      <div className="flex items-center gap-2">
        <ValidationIcon status={milestone.validation_status} />
        <span className="text-sm font-medium">{milestone.title}</span>
        <span
          data-testid={`validation-status-${milestone.id}`}
          className={cn(
            "text-xs",
            milestone.validation_status === "passed" && "text-success",
            milestone.validation_status === "failed" && "text-destructive",
            milestone.validation_status === "pending" && "text-muted-foreground/50",
          )}
        >
          {milestone.validation_status}
        </span>
      </div>
      <MilestoneDoD milestoneId={milestone.id} />
    </div>
  );
}

export function FeatureMilestonesSection({ featureId }: { featureId: string }) {
  const { t } = useT("features");
  const wsId = useWorkspaceId();
  const { data: milestones = [], isLoading } = useQuery(featureMilestonesOptions(wsId, featureId));

  if (isLoading) {
    return (
      <div className="space-y-2">
        <Skeleton className="h-4 w-32" />
        <Skeleton className="h-4 w-48" />
      </div>
    );
  }

  if (milestones.length === 0) {
    return <p className="text-xs text-muted-foreground">{t(($) => $.milestones.empty)}</p>;
  }

  return (
    <div className="space-y-3">
      {milestones.map((m) => (
        <MilestoneRow key={m.id} milestone={m} />
      ))}
    </div>
  );
}
