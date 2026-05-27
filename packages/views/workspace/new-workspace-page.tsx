"use client";

import { ArrowLeft } from "lucide-react";
import { Button } from "@multica/ui/components/ui/button";
import type { Workspace } from "@multica/core/types";
import { useT } from "../i18n";
import { CreateWorkspaceForm } from "./create-workspace-form";

/**
 * Full-page shell for the "create workspace" transition (Next.js route
 * `/workspaces/new`). A Back button is shown when the caller provides
 * `onBack` — i.e. when the user already has at least one workspace.
 */
export function NewWorkspacePage({
  onSuccess,
  onBack,
}: {
  onSuccess: (workspace: Workspace) => void;
  onBack?: () => void;
}) {
  const { t } = useT("workspace");

  return (
    <div className="relative flex min-h-svh flex-col bg-background">
      {onBack && (
        <Button
          variant="ghost"
          size="sm"
          className="absolute top-16 left-12 text-muted-foreground"
          onClick={onBack}
        >
          <ArrowLeft />
          {t(($) => $.new_page.back)}
        </Button>
      )}

      <div className="flex flex-1 flex-col items-center justify-center px-6 pb-12">
        <div className="flex w-full max-w-md flex-col items-center gap-6">
          <div className="text-center">
            <h1 className="text-3xl font-semibold tracking-tight">
              {t(($) => $.new_page.title)}
            </h1>
            <p className="mt-3 text-muted-foreground">
              {t(($) => $.new_page.description)}
            </p>
          </div>
          <CreateWorkspaceForm onSuccess={onSuccess} />
        </div>
      </div>
    </div>
  );
}
