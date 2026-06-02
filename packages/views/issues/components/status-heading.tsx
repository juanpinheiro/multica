import type { IssueStatus } from "@multica/core/types";
import { StatusIcon } from "./status-icon";
import { useT } from "../../i18n";

export function StatusHeading({
  status,
  count,
  liveCount,
}: {
  status: IssueStatus;
  count: number;
  liveCount?: number;
}) {
  const { t } = useT("issues");
  return (
    <div className="flex items-center gap-2">
      <span className="inline-flex items-center gap-1.5 text-xs font-semibold">
        <StatusIcon status={status} className="h-3 w-3" />
        {t(($) => $.status[status])}
      </span>
      <span className="text-xs text-muted-foreground">{count}</span>
      {liveCount !== undefined && liveCount > 0 && (
        <span className="inline-flex items-center gap-1 rounded-full bg-brand/15 px-1.5 py-0.5 text-[10px] font-medium tabular-nums text-brand">
          <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-brand" />
          {liveCount}
        </span>
      )}
    </div>
  );
}
