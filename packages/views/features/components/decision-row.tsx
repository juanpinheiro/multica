"use client";

import type { DecisionLogEntry } from "@multica/core/types";
import { cn } from "@multica/ui/lib/utils";
import { useT } from "../../i18n";

const ADR_PATH_PREFIX = "docs/adr";
const CONTEXT_DOC_PATH = "CONTEXT.md";

function slugifyContextTerm(term: string): string {
  return term
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/(^-|-$)/g, "");
}

function adrHref(ref: string): string {
  return `${ADR_PATH_PREFIX}/${encodeURIComponent(ref)}.md`;
}

function contextTermHref(term: string): string {
  return `${CONTEXT_DOC_PATH}#${slugifyContextTerm(term)}`;
}

function ChipLink({
  href,
  children,
  className,
  testId,
}: {
  href: string;
  children: React.ReactNode;
  className?: string;
  testId?: string;
}) {
  return (
    <a
      href={href}
      data-testid={testId}
      className={cn(
        "inline-flex items-center rounded-sm border px-1 py-px text-[10px] leading-4 text-muted-foreground hover:text-foreground",
        className,
      )}
    >
      {children}
    </a>
  );
}

export interface DecisionRowFeatureChip {
  title: string;
  href: string;
}

export interface DecisionRowProps {
  entry: DecisionLogEntry;
  featureChip?: DecisionRowFeatureChip;
}

export function DecisionRow({ entry, featureChip }: DecisionRowProps) {
  const { t } = useT("features");
  const hasChips =
    entry.adr_refs.length > 0 ||
    entry.context_terms.length > 0 ||
    Boolean(featureChip);

  return (
    <div
      data-testid={`decision-row-${entry.id}`}
      className="space-y-1 border-l-2 border-border pl-3"
    >
      <p className="text-sm font-medium leading-snug">{entry.title}</p>
      <p className="text-xs leading-snug text-foreground/80">{entry.decision}</p>
      {entry.learning ? (
        <p className="text-xs leading-snug text-muted-foreground">
          <span className="font-medium">{t(($) => $.detail.decision_learned)}</span>{" "}
          {entry.learning}
        </p>
      ) : null}
      {hasChips && (
        <div className="flex flex-wrap gap-1 pt-0.5">
          {entry.adr_refs.map((ref) => (
            <ChipLink
              key={`adr-${ref}`}
              href={adrHref(ref)}
              testId={`decision-adr-chip-${ref}`}
              className="border-primary/30 text-primary/80"
            >
              ADR-{ref}
            </ChipLink>
          ))}
          {entry.context_terms.map((term) => (
            <ChipLink
              key={`term-${term}`}
              href={contextTermHref(term)}
              testId={`decision-term-chip-${term}`}
            >
              {term}
            </ChipLink>
          ))}
          {featureChip && (
            <ChipLink
              href={featureChip.href}
              testId="decision-feature-chip"
              className="border-foreground/30 text-foreground/70"
            >
              {featureChip.title}
            </ChipLink>
          )}
        </div>
      )}
    </div>
  );
}
