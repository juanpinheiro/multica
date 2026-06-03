"use client";

import {
  Activity,
  Bell,
  Bot,
  ChevronRight,
  DollarSign,
  GitBranch,
  Layers,
  Plug,
  ScrollText,
  Settings,
  Sparkles,
  Wrench,
} from "lucide-react";
import { INITIATIVES } from "../initiative-runner/_data";
import type { Status } from "../initiative-runner/_data";
import { SharedBody } from "./_shared";

// Chrome B — Wide nav (~260px). Surfaces initiatives directly in the
// sidebar as nav items so the user doesn't need an extra screen to see
// what's in flight. Status dot per item, live dots on running ones,
// section headers for grouping. Replaces "My Issues" / "New Issue" /
// workspace switcher entirely.

const statusDot = (s: Status): string => {
  switch (s) {
    case "running": return "bg-emerald-500 animate-pulse";
    case "in_review": return "bg-indigo-500";
    case "blocked": return "bg-red-500";
    case "done": return "bg-muted-foreground";
    case "ready": return "bg-amber-500";
  }
};

export function ChromeWide() {
  const groups: { label: string; status: Status[] }[] = [
    { label: "Running", status: ["running"] },
    { label: "In review", status: ["in_review"] },
    { label: "Other", status: ["ready", "blocked"] },
    { label: "Done", status: ["done"] },
  ];

  return (
    <div className="flex h-screen">
      {/* Sidebar */}
      <aside className="flex w-64 flex-col border-r border-border bg-card/40">
        {/* Project header */}
        <div className="border-b border-border px-4 py-3">
          <div className="flex items-center gap-2">
            <div className="grid size-7 place-items-center rounded-md bg-foreground text-background">
              <Sparkles className="size-3.5" />
            </div>
            <div className="min-w-0">
              <div className="truncate text-sm font-semibold">~/code/upgrade</div>
              <div className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
                <GitBranch className="size-3" />
                <span>worktree</span>
                <span>·</span>
                <span>3 repos</span>
              </div>
            </div>
          </div>
        </div>

        {/* Primary nav */}
        <nav className="border-b border-border p-2">
          <NavItem icon={Activity} label="Live" active />
          <NavItem icon={Layers} label="Initiatives" count={INITIATIVES.length} />
          <NavItem icon={ScrollText} label="Decisions" />
          <NavItem icon={Bell} label="Inbox" badge="2" />
        </nav>

        {/* Workbench nav — agents/runtimes/skills/costs */}
        <nav className="border-b border-border p-2">
          <SectionLabel>Workbench</SectionLabel>
          <NavItem icon={Bot} label="Agents" count={5} liveDots={4} />
          <NavItem icon={DollarSign} label="Costs" trailing="$12.40" />
          <NavItem icon={Plug} label="Skills" count={12} />
          <NavItem icon={Wrench} label="Runtimes" count={1} statusDot="ok" />
        </nav>

        {/* Initiative list as nav */}
        <div className="flex-1 overflow-y-auto p-2">
          <SectionLabel>Pinned initiatives</SectionLabel>
          {groups.map((g) => {
            const items = INITIATIVES.filter((i) => g.status.includes(i.status));
            if (items.length === 0) return null;
            return (
              <div key={g.label} className="mb-3">
                <SectionLabel>{g.label}</SectionLabel>
                <ul>
                  {items.map((init) => (
                    <li key={init.id}>
                      <button className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm hover:bg-muted">
                        <span className={`size-2 shrink-0 rounded-full ${statusDot(init.status)}`} />
                        <span className="flex-1 truncate text-muted-foreground hover:text-foreground">
                          {init.title}
                        </span>
                        {init.status === "running" && (
                          <span className="text-[10px] text-emerald-500">●●</span>
                        )}
                      </button>
                    </li>
                  ))}
                </ul>
              </div>
            );
          })}
        </div>

        {/* Footer */}
        <div className="border-t border-border p-3 text-[11px] text-muted-foreground">
          <div className="flex items-center justify-between">
            <span className="inline-flex items-center gap-1.5">
              <span className="relative inline-flex size-1.5">
                <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75" />
                <span className="relative inline-flex size-1.5 rounded-full bg-emerald-500" />
              </span>
              4 agents active
            </span>
            <Settings className="size-3.5" />
          </div>
        </div>
      </aside>

      {/* Main */}
      <main className="flex-1 overflow-y-auto">
        <div className="mx-auto max-w-3xl px-8 py-8">
          <SharedBody />
        </div>
      </main>
    </div>
  );
}

function NavItem({
  icon: Icon,
  label,
  count,
  active,
  badge,
  trailing,
  liveDots,
  statusDot,
}: {
  icon: typeof Activity;
  label: string;
  count?: number;
  active?: boolean;
  badge?: string;
  trailing?: string;
  liveDots?: number;
  statusDot?: "ok" | "warn" | "err";
}) {
  return (
    <button
      className={`flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm transition ${
        active ? "bg-muted text-foreground" : "text-muted-foreground hover:bg-muted hover:text-foreground"
      }`}
    >
      <Icon className="size-4" />
      <span className="flex-1 text-left">{label}</span>
      {liveDots ? (
        <span className="inline-flex items-center gap-0.5 text-[11px] text-emerald-500">
          <span className="size-1.5 animate-pulse rounded-full bg-emerald-500" />
          {liveDots}
        </span>
      ) : null}
      {statusDot && (
        <span
          className={`size-1.5 rounded-full ${
            statusDot === "ok" ? "bg-emerald-500" : statusDot === "warn" ? "bg-amber-500" : "bg-red-500"
          }`}
        />
      )}
      {trailing && (
        <span className="text-[11px] tabular-nums text-muted-foreground">{trailing}</span>
      )}
      {badge && (
        <span className="rounded-full bg-foreground px-1.5 py-px text-[10px] font-medium text-background">
          {badge}
        </span>
      )}
      {typeof count === "number" && !badge && (
        <span className="text-[11px] tabular-nums text-muted-foreground">{count}</span>
      )}
      {active && <ChevronRight className="size-3" />}
    </button>
  );
}

function SectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <div className="px-2 py-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
      {children}
    </div>
  );
}
