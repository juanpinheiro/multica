import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { DecisionLogEntry } from "@multica/core/types";
import layout from "../../locales/en/layout.json";
import features from "../../locales/en/features.json";
import { DecisionsPage } from "./decisions-page";

const { decisions, featureList } = vi.hoisted(() => ({
  decisions: { current: [] as DecisionLogEntry[], loading: false },
  featureList: { current: [] as Array<{ id: string; title: string }> },
}));

function pickFromLayout(selector: (m: typeof layout) => string): string {
  return selector(layout);
}

function pickFromFeatures(selector: (m: typeof features) => string): string {
  return selector(features);
}

vi.mock("../../i18n", () => ({
  useT: (ns: "layout" | "features" = "layout") => ({
    t: (selector: (m: typeof layout & typeof features) => string) =>
      ns === "features"
        ? pickFromFeatures(selector as unknown as (m: typeof features) => string)
        : pickFromLayout(selector as unknown as (m: typeof layout) => string),
  }),
}));

vi.mock("@multica/core/hooks", () => ({
  useWorkspaceId: () => "ws-1",
}));

vi.mock("@multica/core/paths", () => ({
  useWorkspacePaths: () => ({
    initiativeDetail: (id: string) => `/acme/initiatives/${id}`,
  }),
}));

vi.mock("@multica/core/decision-log/queries", () => ({
  workspaceDecisionsOptions: () => ({ queryKey: ["decision-log", "workspace"] }),
}));

vi.mock("@multica/core/features/queries", () => ({
  featureListOptions: () => ({ queryKey: ["features"] }),
}));

vi.mock("@multica/ui/components/ui/skeleton", () => ({
  Skeleton: () => <div data-testid="skeleton" />,
}));

vi.mock("@multica/ui/lib/utils", () => ({
  cn: (...args: (string | false | null | undefined)[]) => args.filter(Boolean).join(" "),
}));

vi.mock("@multica/ui/components/ui/sidebar", () => ({
  SidebarTrigger: () => null,
  useSidebar: () => {
    throw new Error("no sidebar provider in test");
  },
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: ({ queryKey }: { queryKey: readonly unknown[] }) => {
    if (queryKey[0] === "decision-log") {
      return { data: decisions.current, isLoading: decisions.loading };
    }
    if (queryKey[0] === "features") {
      return { data: featureList.current, isLoading: false };
    }
    return { data: undefined, isLoading: false };
  },
}));

function makeEntry(id: string, overrides: Partial<DecisionLogEntry> = {}): DecisionLogEntry {
  return {
    id,
    workspace_id: "ws-1",
    feature_id: "f-1",
    run_id: "r-1",
    title: `Decision ${id}`,
    decision: `We decided ${id}`,
    learning: "",
    adr_refs: [],
    context_terms: [],
    created_at: "2026-06-01T00:00:00Z",
    ...overrides,
  };
}

describe("DecisionsPage", () => {
  beforeEach(() => {
    decisions.current = [];
    decisions.loading = false;
    featureList.current = [{ id: "f-1", title: "Refactor checkout" }];
  });

  it("renders the empty state when there are no decisions", () => {
    render(<DecisionsPage />);
    expect(screen.getByTestId("decisions-empty")).toBeInTheDocument();
  });

  it("renders one row per decision", () => {
    decisions.current = [
      makeEntry("d1", { title: "Use splice" }),
      makeEntry("d2", { title: "Adopt zod" }),
    ];
    render(<DecisionsPage />);
    expect(screen.getByTestId("decision-row-d1")).toHaveTextContent("Use splice");
    expect(screen.getByTestId("decision-row-d2")).toHaveTextContent("Adopt zod");
  });

  it("renders an ADR chip as an anchor pointing at docs/adr/<ref>.md", () => {
    decisions.current = [makeEntry("d1", { adr_refs: ["0004"] })];
    render(<DecisionsPage />);
    const chip = screen.getByTestId("decision-adr-chip-0004") as HTMLAnchorElement;
    expect(chip.tagName).toBe("A");
    expect(chip.getAttribute("href")).toBe("docs/adr/0004.md");
  });

  it("renders a context-term chip pointing at CONTEXT.md#<slug>", () => {
    decisions.current = [makeEntry("d1", { context_terms: ["Gate"] })];
    render(<DecisionsPage />);
    const chip = screen.getByTestId("decision-term-chip-Gate") as HTMLAnchorElement;
    expect(chip.getAttribute("href")).toBe("CONTEXT.md#gate");
  });

  it("renders a feature chip that links to the initiative detail path", () => {
    decisions.current = [makeEntry("d1")];
    render(<DecisionsPage />);
    const chip = screen.getByTestId("decision-feature-chip") as HTMLAnchorElement;
    expect(chip.getAttribute("href")).toBe("/acme/initiatives/f-1");
    expect(chip).toHaveTextContent("Refactor checkout");
  });

  it("shows the loading skeleton while the query is pending", () => {
    decisions.loading = true;
    render(<DecisionsPage />);
    expect(screen.getByTestId("decisions-loading")).toBeInTheDocument();
  });
});
