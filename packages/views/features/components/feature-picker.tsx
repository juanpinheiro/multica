"use client";

import { Check, FolderKanban, X } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { featureListOptions } from "@multica/core/features/queries";
import { useWorkspaceId } from "@multica/core/hooks";
import type { UpdateIssueRequest } from "@multica/core/types";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from "@multica/ui/components/ui/dropdown-menu";
import { FeatureIcon } from "./feature-icon";
import { useT } from "../../i18n";

export function FeaturePicker({
  featureId,
  onUpdate,
  triggerRender,
  align = "start",
  defaultOpen = false,
}: {
  featureId: string | null;
  onUpdate: (updates: Partial<UpdateIssueRequest>) => void;
  triggerRender?: React.ReactElement;
  align?: "start" | "center" | "end";
  /** Open the dropdown on first mount. Used by progressive-disclosure
   *  sidebars so a newly-added field immediately enters edit state. */
  defaultOpen?: boolean;
}) {
  const { t } = useT("features");
  const wsId = useWorkspaceId();
  const { data: features = [] } = useQuery(featureListOptions(wsId));
  const current = features.find((p) => p.id === featureId);

  return (
    <DropdownMenu defaultOpen={defaultOpen}>
      <DropdownMenuTrigger
        className={triggerRender ? undefined : "flex items-center gap-1.5 cursor-pointer rounded px-1 -mx-1 hover:bg-accent/30 transition-colors overflow-hidden"}
        render={triggerRender}
      >
        {current ? (
          <FeatureIcon feature={current} size="sm" />
        ) : (
          <FolderKanban className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
        )}
        <span className="truncate">{current ? current.title : t(($) => $.picker.no_feature)}</span>
      </DropdownMenuTrigger>
      <DropdownMenuContent align={align} className="w-52">
        {features.map((p) => (
          <DropdownMenuItem key={p.id} onClick={() => onUpdate({ feature_id: p.id })}>
            <FeatureIcon feature={p} size="md" className="mr-1" />
            <span className="truncate">{p.title}</span>
            {p.id === featureId && <Check className="ml-auto h-3.5 w-3.5 shrink-0" />}
          </DropdownMenuItem>
        ))}
        {features.length > 0 && featureId && <DropdownMenuSeparator />}
        {featureId && (
          <DropdownMenuItem onClick={() => onUpdate({ feature_id: null })}>
            <X className="h-3.5 w-3.5 text-muted-foreground" />
            {t(($) => $.picker.remove)}
          </DropdownMenuItem>
        )}
        {features.length === 0 && (
          <div className="px-2 py-1.5 text-xs text-muted-foreground">{t(($) => $.picker.empty)}</div>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
