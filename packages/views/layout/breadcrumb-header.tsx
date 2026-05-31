"use client";

import type { ReactNode } from "react";
import { Fragment } from "react";
import { cn } from "@multica/ui/lib/utils";
import { PageHeader } from "./page-header";
import { AppLink } from "../navigation/app-link";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@multica/ui/components/ui/breadcrumb";

export interface BreadcrumbSegment {
  label: ReactNode;
  href?: string;
  className?: string;
}

interface BreadcrumbHeaderProps {
  segments: BreadcrumbSegment[];
  actions?: ReactNode;
  className?: string;
}

export function BreadcrumbHeader({
  segments,
  actions,
  className,
}: BreadcrumbHeaderProps) {
  return (
    <PageHeader className={cn("gap-2 bg-background text-sm", className)}>
      <Breadcrumb className="flex-1 min-w-0">
        <BreadcrumbList className="flex-nowrap gap-1 text-sm">
          {segments.map((seg, i) => {
            const isLast = i === segments.length - 1;
            const key = seg.href ?? `segment-${i}`;
            return (
              <Fragment key={key}>
                <BreadcrumbItem>
                  {seg.href ? (
                    <BreadcrumbLink
                      render={
                        <AppLink
                          href={seg.href}
                          className={cn(
                            "text-muted-foreground hover:text-foreground transition-colors shrink-0",
                            seg.className,
                          )}
                        />
                      }
                    >
                      {seg.label}
                    </BreadcrumbLink>
                  ) : isLast ? (
                    <BreadcrumbPage
                      className={cn("truncate font-medium text-foreground", seg.className)}
                    >
                      {seg.label}
                    </BreadcrumbPage>
                  ) : (
                    <span className={cn("truncate text-muted-foreground", seg.className)}>
                      {seg.label}
                    </span>
                  )}
                </BreadcrumbItem>
                {!isLast && <BreadcrumbSeparator className="text-muted-foreground/50" />}
              </Fragment>
            );
          })}
        </BreadcrumbList>
      </Breadcrumb>
      {actions && (
        <div className="flex items-center gap-1 shrink-0">{actions}</div>
      )}
    </PageHeader>
  );
}
