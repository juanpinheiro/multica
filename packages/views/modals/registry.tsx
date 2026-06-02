"use client";

import { useModalStore } from "@multica/core/modals";
import { SetParentIssueModal } from "./set-parent-issue";
import { AddChildIssueModal } from "./add-child-issue";
import { DeleteIssueConfirmModal } from "./delete-issue-confirm";
import { BacklogAgentHintModal } from "./backlog-agent-hint";

export function ModalRegistry() {
  const modal = useModalStore((s) => s.modal);
  const data = useModalStore((s) => s.data);
  const close = useModalStore((s) => s.close);

  switch (modal) {
    case "issue-set-parent":
      return <SetParentIssueModal onClose={close} data={data} />;
    case "issue-add-child":
      return <AddChildIssueModal onClose={close} data={data} />;
    case "issue-delete-confirm":
      return <DeleteIssueConfirmModal onClose={close} data={data} />;
    case "issue-backlog-agent-hint":
      return <BacklogAgentHintModal onClose={close} data={data} />;
    default:
      return null;
  }
}
