"use client";

import {
  Activity,
  GitBranch,
  Layers,
  ScrollText,
  Settings,
  Sparkles,
} from "lucide-react";
import { SharedBody } from "./_shared";

// Chrome A — Thin icon rail (~56px). Manifest-driven app, no workspace
// switcher. Branch+mode indicator at the bottom replaces the workspace
// dropdown. Tooltips on hover surface labels.

type RailItem = {
  key: string;
  icon: typeof Activity;
  label: string;
  active?: boolean;
};
const ITEMS: RailItem[] = [
  { key: "live", icon: Activity, label: "Live", active: true },
  { key: "initiatives", icon: Layers, label: "Initiatives" },
  { key: "decisions", icon: ScrollText, label: "Decisions" },
];

export function ChromeRail() {
  return (
    <div className="flex h-screen">
      {/* Rail */}
      <aside className="flex w-14 flex-col items-center justify-between border-r border-border bg-card/30 py-3">
        <div className="flex flex-col items-center gap-1">
          <div className="mb-3 grid size-8 place-items-center rounded-md bg-foreground text-background">
            <Sparkles className="size-4" />
          </div>
          {ITEMS.map(({ key, icon: Icon, label, active }) => (
            <button
              key={key}
              title={label}
              className={`group relative grid size-9 place-items-center rounded-md transition ${
                active
                  ? "bg-muted text-foreground"
                  : "text-muted-foreground hover:bg-muted hover:text-foreground"
              }`}
            >
              <Icon className="size-4" />
              {active && (
                <span className="absolute left-0 top-1/2 h-5 w-0.5 -translate-y-1/2 rounded-r bg-foreground" />
              )}
              <span className="pointer-events-none absolute left-full ml-3 hidden whitespace-nowrap rounded bg-popover px-2 py-1 text-xs text-popover-foreground shadow group-hover:block">
                {label}
              </span>
            </button>
          ))}
        </div>
        <div className="flex flex-col items-center gap-2 text-muted-foreground">
          <button
            title="Settings"
            className="grid size-9 place-items-center rounded-md hover:bg-muted hover:text-foreground"
          >
            <Settings className="size-4" />
          </button>
          {/* Branch indicator at the bottom of the rail */}
          <div
            title="multica · worktree"
            className="grid size-9 place-items-center rounded-md border border-border bg-card text-[10px] font-mono"
          >
            <GitBranch className="size-3.5" />
          </div>
        </div>
      </aside>

      {/* Main */}
      <main className="flex-1 overflow-y-auto">
        {/* Slim project bar — manifest ambient */}
        <div className="flex items-center justify-between border-b border-border px-6 py-2 text-xs">
          <div className="flex items-center gap-2 text-muted-foreground">
            <GitBranch className="size-3.5" />
            <span className="font-mono">~/code/upgrade</span>
            <span className="rounded border border-border bg-card px-1.5 py-0.5">worktree</span>
          </div>
          <div className="text-muted-foreground">via .multica/workspace.toml</div>
        </div>

        <div className="mx-auto max-w-4xl px-8 py-8">
          <SharedBody />
        </div>
      </main>
    </div>
  );
}
