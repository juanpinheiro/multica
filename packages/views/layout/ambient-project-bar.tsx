"use client";

import { GitBranch, Layers2 } from "lucide-react";
import { cn } from "@multica/ui/lib/utils";
import { useCurrentWorkspace } from "@multica/core/paths";

function ModeTag({ mode }: { mode: "worktree" | "in_place" }) {
  const isInPlace = mode === "in_place";
  return (
    <span
      data-testid="ambient-mode-tag"
      className={cn(
        "inline-flex items-center gap-1 rounded px-1.5 py-0.5 font-mono text-xs",
        isInPlace ? "bg-warning/10 text-warning" : "bg-muted text-muted-foreground",
      )}
    >
      {isInPlace && <Layers2 className="h-3 w-3" />}
      {isInPlace ? "in_place" : "worktree"}
    </span>
  );
}

export function AmbientProjectBar() {
  const workspace = useCurrentWorkspace();
  if (!workspace) return null;

  const mode = workspace.mode ?? "worktree";

  return (
    <div className="flex h-7 shrink-0 items-center gap-2 border-b border-border/50 bg-muted/20 px-4 text-xs text-muted-foreground">
      <span className="font-medium text-foreground">{workspace.name}</span>
      <span aria-hidden>·</span>
      <span
        data-testid="ambient-branch"
        className="inline-flex items-center gap-1 font-mono"
      >
        <GitBranch className="h-3 w-3" />
        {workspace.slug}
      </span>
      <span aria-hidden>·</span>
      <ModeTag mode={mode} />
      <span aria-hidden>·</span>
      <span>via .multica</span>
    </div>
  );
}
