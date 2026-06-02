import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AmbientProjectBar } from "./ambient-project-bar";

const { workspace } = vi.hoisted(() => ({
  workspace: {
    current: null as { name: string; mode?: "worktree" | "in_place" } | null,
  },
}));

vi.mock("@multica/core/paths", () => ({
  useCurrentWorkspace: () => workspace.current,
}));

describe("AmbientProjectBar", () => {
  it("renders nothing when workspace is null", () => {
    workspace.current = null;
    const { container } = render(<AmbientProjectBar />);
    expect(container).toBeEmptyDOMElement();
  });

  it("renders workspace name in worktree mode", () => {
    workspace.current = { name: "Acme", mode: "worktree" };
    render(<AmbientProjectBar />);
    expect(screen.getByText("Acme")).toBeInTheDocument();
    const modeTag = screen.getByTestId("ambient-mode-tag");
    expect(modeTag).toHaveTextContent("worktree");
    expect(modeTag).not.toHaveClass("text-warning");
  });

  it("renders in_place mode with amber styling", () => {
    workspace.current = { name: "Acme", mode: "in_place" };
    render(<AmbientProjectBar />);
    const modeTag = screen.getByTestId("ambient-mode-tag");
    expect(modeTag).toHaveTextContent("in_place");
    expect(modeTag).toHaveClass("text-warning");
  });

  it("treats absent mode as worktree", () => {
    workspace.current = { name: "Acme" };
    render(<AmbientProjectBar />);
    const modeTag = screen.getByTestId("ambient-mode-tag");
    expect(modeTag).toHaveTextContent("worktree");
    expect(modeTag).not.toHaveClass("text-warning");
  });

  it("shows manifest provenance hint", () => {
    workspace.current = { name: "Acme", mode: "worktree" };
    render(<AmbientProjectBar />);
    expect(screen.getByText(/via \.multica/i)).toBeInTheDocument();
  });
});
