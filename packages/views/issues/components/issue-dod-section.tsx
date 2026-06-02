"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ChevronRight } from "lucide-react";
import type { DodAssertion } from "@multica/core/types";
import { issueDodOptions } from "@multica/core/dod/queries";
import { cn } from "@multica/ui/lib/utils";
import { useT } from "../../i18n";

interface IssueDodSectionProps {
  issueId: string;
  wsId: string;
}

function assertionMarker(status: DodAssertion["status"]): string {
  if (status === "passed") return "✓";
  if (status === "failed") return "✗";
  return "○";
}

function assertionTextClass(status: DodAssertion["status"]): string {
  if (status === "passed") return "text-success";
  if (status === "failed") return "text-destructive";
  return "text-muted-foreground/60";
}

export function IssueDodSection({ issueId, wsId }: IssueDodSectionProps) {
  const { t } = useT("issues");
  const [open, setOpen] = useState(true);
  const { data: assertions = [] } = useQuery(issueDodOptions(wsId, issueId));

  if (assertions.length === 0) return null;

  return (
    <div>
      <button
        type="button"
        className={`flex w-full items-center gap-1 rounded-md px-2 py-1 text-xs font-medium transition-colors mb-2 hover:bg-accent/70 ${
          open ? "" : "text-muted-foreground hover:text-foreground"
        }`}
        onClick={() => setOpen(!open)}
      >
        {t(($) => $.detail.section_acceptance_criteria)}
        <ChevronRight
          className={`!size-3 shrink-0 stroke-[2.5] text-muted-foreground transition-transform ${
            open ? "rotate-90" : ""
          }`}
        />
      </button>
      {open && (
        <div className="space-y-0.5 pl-2">
          {assertions.map((a) => (
            <div key={a.id} className="flex items-start gap-1.5">
              <span
                data-testid={`dod-marker-${a.id}`}
                className={cn("shrink-0 text-[10px] leading-4", assertionTextClass(a.status))}
              >
                {assertionMarker(a.status)}
              </span>
              <span className={cn("text-xs leading-snug", assertionTextClass(a.status))}>
                {a.text}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
