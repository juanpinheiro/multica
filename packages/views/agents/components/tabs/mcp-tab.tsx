"use client";

import { useEffect, useState } from "react";
import { Loader2, Save } from "lucide-react";
import type { Agent } from "@multica/core/types";
import { Button } from "@multica/ui/components/ui/button";
import { Textarea } from "@multica/ui/components/ui/textarea";
import { toast } from "sonner";
import { useT } from "../../../i18n";

function serializeConfig(config: Record<string, unknown> | null | undefined): string {
  if (!config) return "";
  return JSON.stringify(config, null, 2);
}

type ParseResult =
  | { value: Record<string, unknown> | null; error: null }
  | { value: null; error: "invalid_json" | "invalid_schema" };

function parseConfig(raw: string): ParseResult {
  const trimmed = raw.trim();
  if (!trimmed) return { value: null, error: null };
  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    return { value: null, error: "invalid_json" };
  }
  if (
    typeof parsed !== "object" ||
    parsed === null ||
    Array.isArray(parsed) ||
    !("mcpServers" in parsed) ||
    typeof (parsed as Record<string, unknown>).mcpServers !== "object" ||
    Array.isArray((parsed as Record<string, unknown>).mcpServers)
  ) {
    return { value: null, error: "invalid_schema" };
  }
  return { value: parsed as Record<string, unknown>, error: null };
}

export function McpTab({
  agent,
  onSave,
  onDirtyChange,
}: {
  agent: Agent;
  onSave: (updates: Record<string, unknown>) => Promise<void>;
  onDirtyChange?: (dirty: boolean) => void;
}) {
  const { t } = useT("agents");
  const original = serializeConfig(agent.mcp_config);
  const [draft, setDraft] = useState(original);
  const [saving, setSaving] = useState(false);

  const { value: parsed, error: parseError } = parseConfig(draft);
  const dirty = draft !== original;
  const canSave = dirty && parseError === null;

  useEffect(() => {
    onDirtyChange?.(dirty);
  }, [dirty, onDirtyChange]);

  if (agent.mcp_config_redacted) {
    return (
      <p className="text-sm text-muted-foreground">
        {t(($) => $.tab_body.mcp.redacted_notice)}
      </p>
    );
  }

  const handleSave = async () => {
    if (!canSave) return;
    setSaving(true);
    try {
      await onSave({ mcp_config: parsed });
      toast.success(t(($) => $.tab_body.mcp.saved_toast));
    } catch (err) {
      toast.error(
        err instanceof Error && err.message
          ? err.message
          : t(($) => $.tab_body.mcp.save_failed_toast),
      );
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-4">
      <p className="text-xs text-muted-foreground">
        {t(($) => $.tab_body.mcp.intro)}
      </p>
      {!agent.mcp_config && (
        <p className="text-xs italic text-muted-foreground">
          {t(($) => $.tab_body.mcp.empty_state)}
        </p>
      )}
      <div className="space-y-1.5">
        <label className="text-xs font-medium">
          {t(($) => $.tab_body.mcp.label)}
        </label>
        <Textarea
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          className="min-h-48 font-mono text-xs"
          placeholder={t(($) => $.tab_body.mcp.placeholder)}
          aria-invalid={parseError !== null && draft.trim().length > 0}
        />
        {parseError !== null && draft.trim().length > 0 && (
          <p className="text-xs text-destructive" role="alert">
            {t(($) => $.tab_body.mcp[parseError])}
          </p>
        )}
      </div>
      <div className="flex items-center justify-end gap-3">
        {dirty && (
          <span className="text-xs text-muted-foreground">
            {t(($) => $.tab_body.common.unsaved_changes)}
          </span>
        )}
        <Button onClick={handleSave} disabled={!canSave || saving} size="sm">
          {saving ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <Save className="h-3.5 w-3.5" />
          )}
          {t(($) => $.tab_body.common.save)}
        </Button>
      </div>
    </div>
  );
}
