"use client";

import type { Handoff } from "@multica/core/types";
import { cn } from "@multica/ui/lib/utils";
import { useT } from "../../i18n";


interface RunHandoffPanelProps {
  handoff: Handoff;
}

export function RunHandoffPanel({ handoff }: RunHandoffPanelProps) {
  const { t } = useT("issues");

  const hasContent =
    handoff.done.length > 0 ||
    handoff.left_undone.length > 0 ||
    handoff.commands.length > 0 ||
    handoff.discoveries.length > 0;

  if (!hasContent) return null;

  return (
    <div className="mt-1.5 space-y-2 pl-2 pr-1 text-xs">
      {handoff.done.length > 0 && (
        <HandoffSubsection label={t(($) => $.execution_log.handoff_done)}>
          {handoff.done.map((item, i) => (
            <ItemRow key={i} text={item} variant="done" />
          ))}
        </HandoffSubsection>
      )}
      {handoff.left_undone.length > 0 && (
        <HandoffSubsection label={t(($) => $.execution_log.handoff_left_undone)}>
          {handoff.left_undone.map((item, i) => (
            <ItemRow key={i} text={item} variant="undone" />
          ))}
        </HandoffSubsection>
      )}
      {handoff.commands.length > 0 && (
        <HandoffSubsection label={t(($) => $.execution_log.handoff_commands)}>
          {handoff.commands.map((cmd, i) => (
            <CommandRow key={i} command={cmd.command} exitCode={cmd.exit_code} />
          ))}
        </HandoffSubsection>
      )}
      {handoff.discoveries.length > 0 && (
        <HandoffSubsection label={t(($) => $.execution_log.handoff_discoveries)}>
          {handoff.discoveries.map((item, i) => (
            <ItemRow key={i} text={item} variant="discovery" />
          ))}
        </HandoffSubsection>
      )}
    </div>
  );
}

function HandoffSubsection({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="space-y-0.5">
      <p className="text-[10px] font-medium uppercase tracking-wide text-muted-foreground/60">
        {label}
      </p>
      <div className="space-y-0.5">{children}</div>
    </div>
  );
}

function ItemRow({
  text,
  variant,
}: {
  text: string;
  variant: "done" | "undone" | "discovery";
}) {
  return (
    <div className="flex items-start gap-1.5">
      <span
        className={cn(
          "shrink-0 text-[10px] leading-4",
          variant === "done" && "text-success",
          variant === "undone" && "text-destructive",
          variant === "discovery" && "text-muted-foreground",
        )}
      >
        {variant === "done" ? "✓" : variant === "undone" ? "✗" : "→"}
      </span>
      <span className="leading-snug text-muted-foreground">{text}</span>
    </div>
  );
}

function CommandRow({ command, exitCode }: { command: string; exitCode: number }) {
  const { t } = useT("issues");
  const ok = exitCode === 0;
  return (
    <div className="flex items-center gap-2">
      <span className="min-w-0 flex-1 truncate font-mono text-[10px] text-muted-foreground">
        {command}
      </span>
      <span
        className={cn(
          "shrink-0 font-mono text-[10px]",
          ok ? "text-success" : "text-destructive",
        )}
      >
        {ok
          ? t(($) => $.execution_log.handoff_exit_ok)
          : t(($) => $.execution_log.handoff_exit_fail, { code: exitCode })}
      </span>
    </div>
  );
}
