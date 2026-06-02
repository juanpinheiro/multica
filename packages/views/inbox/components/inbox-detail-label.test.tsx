import { describe, it, expect, vi } from "vitest";
import { screen } from "@testing-library/react";
import type { InboxItem } from "@multica/core/types";
import { renderWithI18n } from "../../test/i18n";

vi.mock("@multica/core/issues/config", () => ({
  STATUS_CONFIG: {},
  PRIORITY_CONFIG: {},
}));

vi.mock("@multica/core/workspace/hooks", () => ({
  useActorName: () => ({ getActorName: vi.fn(() => "Agent Name") }),
}));

vi.mock("../../issues/components", () => ({
  StatusIcon: ({ status }: { status: string }) => <span data-testid={`status-${status}`} />,
  PriorityIcon: ({ priority }: { priority: string }) => <span data-testid={`priority-${priority}`} />,
}));

vi.mock("./inbox-display", () => ({
  getQuickCreateFailureDetail: () => "",
}));

function item(overrides: Partial<InboxItem>): InboxItem {
  return {
    id: "inbox-1",
    workspace_id: "workspace-1",
    recipient_type: "member",
    recipient_id: "member-1",
    actor_type: "system",
    actor_id: null,
    type: "new_comment",
    severity: "info",
    issue_id: null,
    title: "Test title",
    body: null,
    issue_status: null,
    read: false,
    archived: false,
    created_at: "2026-04-29T12:00:00Z",
    details: null,
    ...overrides,
  };
}

import { InboxDetailLabel } from "./inbox-detail-label";

describe("InboxDetailLabel — initiative types", () => {
  it("renders tripwire reason label for failure_tolerance", () => {
    renderWithI18n(
      <InboxDetailLabel
        item={item({
          type: "initiative_tripwire",
          details: { feature_id: "f1", reason: "failure_tolerance", mode: "afk" },
        })}
      />,
    );
    expect(screen.getByText(/failure/i)).toBeInTheDocument();
  });

  it("renders a distinct label for each tripwire reason", () => {
    const { rerender } = renderWithI18n(
      <InboxDetailLabel
        item={item({
          type: "initiative_tripwire",
          details: { reason: "token_budget", feature_id: "f1", mode: "afk" },
        })}
      />,
    );
    expect(screen.getByText(/token/i)).toBeInTheDocument();

    rerender(
      <InboxDetailLabel
        item={item({
          type: "initiative_tripwire",
          details: { reason: "run_budget", feature_id: "f1", mode: "hitl" },
        })}
      />,
    );
    expect(screen.getByText(/run/i)).toBeInTheDocument();
  });

  it("renders branch slug for feature_ready_for_review", () => {
    renderWithI18n(
      <InboxDetailLabel
        item={item({
          type: "feature_ready_for_review",
          details: { feature_id: "f1", branch_slug: "auth-v2", pr_url: "" },
        })}
      />,
    );
    expect(screen.getByText(/auth-v2/)).toBeInTheDocument();
  });

  it("falls back to type label when branch_slug is missing for feature_ready_for_review", () => {
    renderWithI18n(
      <InboxDetailLabel
        item={item({
          type: "feature_ready_for_review",
          details: { feature_id: "f1" },
        })}
      />,
    );
    expect(screen.getByText(/ready for review/i)).toBeInTheDocument();
  });

  it("renders branch prompt for feature_pr_draft", () => {
    renderWithI18n(
      <InboxDetailLabel
        item={item({
          type: "feature_pr_draft",
          details: { feature_id: "f1", branch_slug: "my-feature" },
        })}
      />,
    );
    expect(screen.getByText(/my-feature/)).toBeInTheDocument();
  });
});
