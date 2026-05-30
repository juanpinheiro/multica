"use client";

import { useQuery } from "@tanstack/react-query";
import { FolderGit } from "lucide-react";
import { Card, CardContent } from "@multica/ui/components/ui/card";
import { useWorkspaceId } from "@multica/core/hooks";
import { repoListOptions } from "@multica/core/workspace/queries";
import { useT } from "../../i18n";

// Read-only view of the workspace's first-class repos. Repos are registered
// through the local setup skill / CLI manifest, not a web form, so this tab
// surfaces what the daemon will check out without offering inline editing.
export function RepositoriesTab() {
  const { t } = useT("settings");
  const wsId = useWorkspaceId();
  const { data: repos = [] } = useQuery(repoListOptions(wsId));

  return (
    <div className="space-y-8">
      <section className="space-y-4">
        <h2 className="text-sm font-semibold">{t(($) => $.repositories.section_title)}</h2>

        <Card>
          <CardContent className="space-y-3">
            <p className="text-xs text-muted-foreground">
              {t(($) => $.repositories.description)}
            </p>

            {repos.length === 0 ? (
              <p className="text-xs text-muted-foreground italic">
                {t(($) => $.repositories.empty)}
              </p>
            ) : (
              <ul className="space-y-2">
                {repos.map((repo) => (
                  <li
                    key={repo.id}
                    className="flex items-start gap-2 rounded-md border bg-muted/50 px-3 py-2"
                  >
                    <FolderGit className="mt-0.5 size-3.5 shrink-0 text-muted-foreground" />
                    <div className="min-w-0">
                      <div className="truncate text-xs font-medium" title={repo.name}>
                        {repo.name}
                      </div>
                      <div
                        className="truncate font-mono text-xs text-muted-foreground"
                        title={repo.remote_url}
                      >
                        {repo.remote_url}
                      </div>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>
      </section>
    </div>
  );
}
