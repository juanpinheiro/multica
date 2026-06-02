"use client";

import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Bot } from "lucide-react";
import { useWorkspaceId } from "@multica/core/hooks";
import { agentListOptions } from "@multica/core/workspace/queries";
import { ActorAvatar } from "../../../common/actor-avatar";
import {
  PropertyPicker,
  PickerItem,
  PickerSection,
  PickerEmpty,
} from "../../../issues/components/pickers/property-picker";
import { useT } from "../../../i18n";
import { matchesPinyin } from "../../../editor/extensions/pinyin-match";

export interface AssigneeSelection {
  type: "agent";
  id: string;
}

export function AgentPicker({
  assignee,
  onChange,
  trigger: customTrigger,
  triggerRender,
  align = "start",
}: {
  assignee: AssigneeSelection | null;
  onChange: (next: AssigneeSelection) => void;
  trigger?: React.ReactNode;
  triggerRender?: React.ReactElement;
  align?: "start" | "center" | "end";
}) {
  const { t } = useT("autopilots");
  const wsId = useWorkspaceId();
  const [open, setOpen] = useState(false);
  const [filter, setFilter] = useState("");
  const { data: agents = [] } = useQuery(agentListOptions(wsId));

  const activeAgents = useMemo(() => agents.filter((a) => !a.archived_at), [agents]);

  const selectedAgent =
    assignee?.type === "agent" ? activeAgents.find((a) => a.id === assignee.id) : undefined;
  const selectedName = selectedAgent?.name;

  const query = filter.trim().toLowerCase();
  const matches = (name: string) =>
    !query || name.toLowerCase().includes(query) || matchesPinyin(name, query);
  const filteredAgents = activeAgents.filter((a) => matches(a.name));

  const isSelected = (id: string) =>
    assignee?.type === "agent" && assignee?.id === id;

  const handlePick = (id: string) => {
    onChange({ type: "agent", id });
    setOpen(false);
  };

  return (
    <PropertyPicker
      open={open}
      onOpenChange={setOpen}
      width="w-56"
      align={align}
      searchable
      searchPlaceholder={t(($) => $.agent_picker.filter_placeholder)}
      onSearchChange={setFilter}
      triggerRender={triggerRender}
      trigger={
        customTrigger ?? (
          <>
            {assignee && selectedAgent ? (
              <>
                <ActorAvatar
                  actorType={assignee.type}
                  actorId={assignee.id}
                  size={16}
                  showStatusDot
                />
                <span className="truncate">{selectedName}</span>
              </>
            ) : (
              <>
                <Bot className="size-3" />
                <span>{t(($) => $.agent_picker.select_assignee)}</span>
              </>
            )}
          </>
        )
      }
    >
      {filteredAgents.length === 0 ? (
        <PickerEmpty />
      ) : (
        <PickerSection label={t(($) => $.agent_picker.agents_group)}>
          {filteredAgents.map((a) => (
            <PickerItem
              key={a.id}
              selected={isSelected(a.id)}
              onClick={() => handlePick(a.id)}
            >
              <ActorAvatar actorType="agent" actorId={a.id} size={16} showStatusDot />
              <span className="truncate">{a.name}</span>
            </PickerItem>
          ))}
        </PickerSection>
      )}
    </PropertyPicker>
  );
}
