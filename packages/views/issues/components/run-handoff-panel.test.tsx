import { describe, it, expect, vi } from "vitest";
import { screen } from "@testing-library/react";
import type { Handoff } from "@multica/core/types";
import { renderWithI18n } from "../../test/i18n";

vi.mock("@multica/ui/lib/utils", () => ({
  cn: (...args: (string | false | null | undefined)[]) => args.filter(Boolean).join(" "),
}));

function makeHandoff(overrides: Partial<Handoff> = {}): Handoff {
  return {
    id: "h1",
    workspace_id: "ws-1",
    issue_id: "issue-1",
    run_id: "run-1",
    done: [],
    left_undone: [],
    commands: [],
    discoveries: [],
    created_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

import { RunHandoffPanel } from "./run-handoff-panel";

describe("RunHandoffPanel", () => {
  it("returns null when handoff has no content", () => {
    const { container } = renderWithI18n(<RunHandoffPanel handoff={makeHandoff()} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("renders done items", () => {
    renderWithI18n(
      <RunHandoffPanel
        handoff={makeHandoff({ done: ["Fixed the bug", "Added tests"] })}
      />,
    );
    expect(screen.getByText("Fixed the bug")).toBeInTheDocument();
    expect(screen.getByText("Added tests")).toBeInTheDocument();
  });

  it("renders left_undone items", () => {
    renderWithI18n(
      <RunHandoffPanel handoff={makeHandoff({ left_undone: ["Review needed"] })} />,
    );
    expect(screen.getByText("Review needed")).toBeInTheDocument();
  });

  it("renders commands", () => {
    renderWithI18n(
      <RunHandoffPanel
        handoff={makeHandoff({ commands: [{ command: "make test", exit_code: 0 }] })}
      />,
    );
    expect(screen.getByText("make test")).toBeInTheDocument();
  });

  it("renders discoveries", () => {
    renderWithI18n(
      <RunHandoffPanel handoff={makeHandoff({ discoveries: ["Found a related bug"] })} />,
    );
    expect(screen.getByText("Found a related bug")).toBeInTheDocument();
  });

  it("does not show done section label when done is empty", () => {
    renderWithI18n(
      <RunHandoffPanel handoff={makeHandoff({ left_undone: ["Something"] })} />,
    );
    expect(screen.queryByText("Done")).not.toBeInTheDocument();
  });

  it("shows exit code for a failed command", () => {
    renderWithI18n(
      <RunHandoffPanel
        handoff={makeHandoff({ commands: [{ command: "make build", exit_code: 1 }] })}
      />,
    );
    expect(screen.getByText("make build")).toBeInTheDocument();
    expect(screen.getByText("exit 1")).toBeInTheDocument();
  });

  it("shows exit 0 for a successful command", () => {
    renderWithI18n(
      <RunHandoffPanel
        handoff={makeHandoff({ commands: [{ command: "pnpm test", exit_code: 0 }] })}
      />,
    );
    expect(screen.getByText("exit 0")).toBeInTheDocument();
  });
});
