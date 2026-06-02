import { describe, it, expect, vi } from "vitest";
import { screen } from "@testing-library/react";
import type { DodAssertion } from "@multica/core/types";
import { renderWithI18n } from "../../test/i18n";

vi.mock("@multica/ui/lib/utils", () => ({
  cn: (...args: (string | false | null | undefined)[]) => args.filter(Boolean).join(" "),
}));

function makeAssertion(
  id: string,
  text: string,
  status: DodAssertion["status"],
): DodAssertion {
  return {
    id,
    workspace_id: "ws-1",
    feature_id: "f1",
    milestone_id: "m1",
    text,
    position: 1,
    created_at: "2026-01-01T00:00:00Z",
    status,
    detail: "",
  };
}

const mockAssertions = vi.hoisted((): { value: DodAssertion[] } => ({ value: [] }));

vi.mock("@tanstack/react-query", async () => {
  const actual = await vi.importActual<typeof import("@tanstack/react-query")>(
    "@tanstack/react-query",
  );
  return {
    ...actual,
    useQuery: () => ({ data: mockAssertions.value }),
  };
});

import { IssueDodSection } from "./issue-dod-section";

describe("IssueDodSection", () => {
  it("returns null when there are no assertions", () => {
    mockAssertions.value = [];
    const { container } = renderWithI18n(
      <IssueDodSection issueId="i1" wsId="ws-1" />,
    );
    expect(container).toBeEmptyDOMElement();
  });

  it("renders assertion text", () => {
    mockAssertions.value = [makeAssertion("a1", "Tests pass", "pending")];
    renderWithI18n(<IssueDodSection issueId="i1" wsId="ws-1" />);
    expect(screen.getByText("Tests pass")).toBeInTheDocument();
  });

  it("renders multiple assertions", () => {
    mockAssertions.value = [
      makeAssertion("a1", "Tests pass", "passed"),
      makeAssertion("a2", "No regressions", "failed"),
    ];
    renderWithI18n(<IssueDodSection issueId="i1" wsId="ws-1" />);
    expect(screen.getByText("Tests pass")).toBeInTheDocument();
    expect(screen.getByText("No regressions")).toBeInTheDocument();
  });

  it("shows passed marker for passed assertions", () => {
    mockAssertions.value = [makeAssertion("a1", "Tests pass", "passed")];
    renderWithI18n(<IssueDodSection issueId="i1" wsId="ws-1" />);
    expect(screen.getByTestId("dod-marker-a1")).toHaveTextContent("✓");
  });

  it("shows failed marker for failed assertions", () => {
    mockAssertions.value = [makeAssertion("a1", "Tests pass", "failed")];
    renderWithI18n(<IssueDodSection issueId="i1" wsId="ws-1" />);
    expect(screen.getByTestId("dod-marker-a1")).toHaveTextContent("✗");
  });

  it("shows pending marker for pending assertions", () => {
    mockAssertions.value = [makeAssertion("a1", "Tests pass", "pending")];
    renderWithI18n(<IssueDodSection issueId="i1" wsId="ws-1" />);
    expect(screen.getByTestId("dod-marker-a1")).toHaveTextContent("○");
  });
});
