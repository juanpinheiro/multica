"use client";

import {
  Activity,
  Cog,
  GitBranch,
  Layers,
  ScrollText,
  Sparkles,
} from "lucide-react";
import { SharedBody } from "./_shared";

// Chrome C — Top horizontal nav, no left rail. Maximises horizontal
// space for the board / feed. The manifest project sits as an ambient
// chip on the left of the bar; live agent count sits on the right as a
// status indicator.

type Tab = { key: string; label: string; icon: typeof Activity; active?: boolean };
const TABS: Tab[] = [
  { key: "live", label: "Live", icon: Activity, active: true },
  { key: "initiatives", label: "Initiatives", icon: Layers },
  { key: "decisions", label: "Decisions", icon: ScrollText },
];

export function ChromeTopNav() {
  return (
    <div className="flex h-screen flex-col">
      {/* Top bar */}
      <header className="flex items-center justify-between border-b border-border bg-card/30 px-4 py-2">
        {/* Logo + project chip */}
        <div className="flex items-center gap-3">
          <div className="grid size-7 place-items-center rounded-md bg-foreground text-background">
            <Sparkles className="size-3.5" />
          </div>
          <div className="flex items-center gap-1.5 rounded-md border border-border bg-card px-2 py-1 text-xs">
            <GitBranch className="size-3" />
            <span className="font-mono">~/code/upgrade</span>
            <span className="text-muted-foreground">·</span>
            <span className="text-muted-foreground">worktree</span>
          </div>
        </div>

        {/* Centre tabs */}
        <nav className="flex items-center gap-1 rounded-full border border-border bg-card p-1">
          {TABS.map(({ key, label, icon: Icon, active }) => (
            <button
              key={key}
              className={`flex items-center gap-1.5 rounded-full px-3 py-1 text-sm transition ${
                active
                  ? "bg-foreground text-background"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              <Icon className="size-3.5" />
              {label}
            </button>
          ))}
        </nav>

        {/* Right cluster */}
        <div className="flex items-center gap-3 text-xs text-muted-foreground">
          <span className="inline-flex items-center gap-1.5">
            <span className="relative inline-flex size-2">
              <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75" />
              <span className="relative inline-flex size-2 rounded-full bg-emerald-500" />
            </span>
            4 agents
          </span>
          <button title="Settings" className="grid size-7 place-items-center rounded-md hover:bg-muted hover:text-foreground">
            <Cog className="size-4" />
          </button>
        </div>
      </header>

      {/* Main */}
      <main className="flex-1 overflow-y-auto">
        <div className="mx-auto max-w-5xl px-10 py-10">
          <SharedBody />
        </div>
      </main>
    </div>
  );
}
