import { describe, it, expect, vi } from "vitest";
import { screen } from "@testing-library/react";
import type { DecisionLogEntry } from "@multica/core/types";
import { renderWithI18n } from "../../test/i18n";

vi.mock("@multica/ui/components/ui/skeleton", () => ({
  Skeleton: () => <div data-testid="skeleton" />,
}));

vi.mock("@multica/ui/lib/utils", () => ({
  cn: (...args: (string | false | null | undefined)[]) => args.filter(Boolean).join(" "),
}));

function makeEntry(id: string, over: Partial<DecisionLogEntry> = {}): DecisionLogEntry {
  return {
    id,
    workspace_id: "ws-1",
    feature_id: "f1",
    run_id: "r1",
    title: `Decision ${id}`,
    decision: `We decided ${id}`,
    learning: `We learned ${id}`,
    adr_refs: [],
    context_terms: [],
    created_at: "2026-01-01T00:00:00Z",
    ...over,
  };
}

const mockDecisions = vi.hoisted((): { value: DecisionLogEntry[]; loading: boolean } => ({
  value: [],
  loading: false,
}));

vi.mock("@tanstack/react-query", async () => {
  const actual = await vi.importActual<typeof import("@tanstack/react-query")>("@tanstack/react-query");
  return {
    ...actual,
    useQuery: (opts: { queryKey: unknown[] }) => {
      const key = opts.queryKey as string[];
      if (key[0] === "decision-log") {
        return { data: mockDecisions.value, isLoading: mockDecisions.loading };
      }
      return { data: undefined, isLoading: false };
    },
  };
});

import { DecisionLogSection } from "./decision-log-section";

describe("DecisionLogSection", () => {
  it("shows a skeleton while loading", () => {
    mockDecisions.value = [];
    mockDecisions.loading = true;
    renderWithI18n(<DecisionLogSection featureId="f1" />);
    expect(screen.getAllByTestId("skeleton").length).toBeGreaterThan(0);
    mockDecisions.loading = false;
  });

  it("shows empty message when no decisions exist", () => {
    mockDecisions.value = [];
    renderWithI18n(<DecisionLogSection featureId="f1" />);
    expect(
      screen.getByText(/No decisions recorded yet/i),
    ).toBeInTheDocument();
  });

  it("renders title, decision and learning for each entry", () => {
    mockDecisions.value = [
      makeEntry("d1", { title: "Use splice", decision: "splice over reassignment", learning: "ids stay stable" }),
    ];
    renderWithI18n(<DecisionLogSection featureId="f1" />);
    expect(screen.getByText("Use splice")).toBeInTheDocument();
    expect(screen.getByText("splice over reassignment")).toBeInTheDocument();
    expect(screen.getByText(/ids stay stable/)).toBeInTheDocument();
  });

  it("renders ADR refs and context terms as chips", () => {
    mockDecisions.value = [makeEntry("d1", { adr_refs: ["0004"], context_terms: ["Gate"] })];
    renderWithI18n(<DecisionLogSection featureId="f1" />);
    expect(screen.getByText("ADR-0004")).toBeInTheDocument();
    expect(screen.getByText("Gate")).toBeInTheDocument();
  });

  it("renders a testid row per decision, newest order preserved", () => {
    mockDecisions.value = [makeEntry("d1"), makeEntry("d2")];
    renderWithI18n(<DecisionLogSection featureId="f1" />);
    const row1 = screen.getByTestId("decision-row-d1");
    const row2 = screen.getByTestId("decision-row-d2");
    expect(row1.compareDocumentPosition(row2)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
  });

  it("omits the learning line when learning is empty", () => {
    mockDecisions.value = [makeEntry("d1", { learning: "" })];
    renderWithI18n(<DecisionLogSection featureId="f1" />);
    expect(screen.queryByText("Learned:")).not.toBeInTheDocument();
  });
});
