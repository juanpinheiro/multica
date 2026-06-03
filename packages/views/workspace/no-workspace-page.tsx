"use client";

import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuthStore } from "@multica/core/auth";
import { paths } from "@multica/core/paths";
import { workspaceListOptions } from "@multica/core/workspace/queries";
import { useNavigation } from "../navigation";
import { useT } from "../i18n";

/**
 * Rendered at `/` when no workspace can be resolved. In the manifest-driven
 * model, workspaces are not created from the web UI — they're written by
 * `/setup-multica` running inside Claude Code. This page tells the user to
 * run that command.
 *
 * Self-healing: if the workspace list query reveals an existing workspace
 * (e.g. the cookie was cleared but the workspace still exists), redirect to
 * the first workspace's Live view instead of showing the empty state.
 */
export function NoWorkspacePage() {
  const { t } = useT("workspace");
  const nav = useNavigation();
  const user = useAuthStore((s) => s.user);
  const { data: workspaces } = useQuery({
    ...workspaceListOptions(),
    enabled: !!user,
  });

  useEffect(() => {
    const first = workspaces?.[0];
    if (first) nav.replace(paths.workspace(first.slug).live());
  }, [workspaces, nav]);

  return (
    <div className="flex min-h-svh flex-col bg-background">
      <div className="flex flex-1 flex-col items-center justify-center gap-6 px-6 pb-12 text-center">
        <div className="max-w-md space-y-3">
          <h1 className="text-2xl font-semibold tracking-tight">
            {t(($) => $.no_workspace.title)}
          </h1>
          <p className="text-muted-foreground">
            {t(($) => $.no_workspace.description)}
          </p>
        </div>
        <div className="flex w-full max-w-md flex-col items-center gap-3">
          <p className="text-sm text-muted-foreground">
            {t(($) => $.no_workspace.command_label)}
          </p>
          <code className="block w-full rounded-md border bg-muted px-4 py-2 text-center font-mono text-sm">
            {t(($) => $.no_workspace.command)}
          </code>
        </div>
        <a
          href="https://github.com/multica-ai/multica/blob/main/docs/adr/0008-standalone-personal-install.md"
          target="_blank"
          rel="noopener noreferrer"
          className="text-sm text-muted-foreground underline-offset-4 hover:underline"
        >
          {t(($) => $.no_workspace.docs_link)}
        </a>
      </div>
    </div>
  );
}
