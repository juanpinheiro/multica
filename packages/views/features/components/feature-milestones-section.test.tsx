import { describe, it, expect, vi } from "vitest";
import { screen } from "@testing-library/react";
import type { DodAssertion, Milestone } from "@multica/core/types";
import { renderWithI18n } from "../../test/i18n";

vi.mock("@multica/core/hooks", () => ({
  useWorkspaceId: () => "ws-1",
}));

vi.mock("@multica/ui/components/ui/skeleton", () => ({
  Skeleton: () => <div data-testid="skeleton" />,
}));

vi.mock("@multica/ui/lib/utils", () => ({
  cn: (...args: (string | false | null | undefined)[]) => args.filter(Boolean).join(" "),
}));

function makeM(
  id: string,
  title: string,
  position: number,
  validation_status: Milestone["validation_status"],
): Milestone {
  return {
    id,
    workspace_id: "ws-1",
    feature_id: "f1",
    title,
    position,
    validation_status,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  };
}

function makeAssertion(
  id: string,
  milestoneId: string,
  text: string,
  status: DodAssertion["status"],
): DodAssertion {
  return {
    id,
    workspace_id: "ws-1",
    feature_id: "f1",
    milestone_id: milestoneId,
    text,
    position: 1,
    created_at: "2026-01-01T00:00:00Z",
    status,
    detail: "",
  };
}

const mockMilestones = vi.hoisted((): { value: Milestone[] } => ({ value: [] }));
const mockDodAssertions = vi.hoisted((): { value: Record<string, DodAssertion[]> } => ({ value: {} }));

vi.mock("@tanstack/react-query", async () => {
  const actual = await vi.importActual<typeof import("@tanstack/react-query")>("@tanstack/react-query");
  return {
    ...actual,
    useQuery: (opts: { queryKey: unknown[] }) => {
      const key = opts.queryKey as string[];
      if (key[0] === "milestones" && key[2] === "feature") {
        return { data: mockMilestones.value, isLoading: false };
      }
      if (key[0] === "dod" && key[2] === "milestone") {
        const milestoneId = key[3] as string;
        return { data: mockDodAssertions.value[milestoneId] ?? [], isLoading: false };
      }
      return { data: undefined, isLoading: false };
    },
  };
});

import { FeatureMilestonesSection } from "./feature-milestones-section";

describe("FeatureMilestonesSection", () => {
  it("shows empty message when no milestones exist", () => {
    mockMilestones.value = [];
    renderWithI18n(<FeatureMilestonesSection featureId="f1" />);
    expect(screen.getByText("No milestones yet")).toBeInTheDocument();
  });

  it("renders each milestone title", () => {
    mockMilestones.value = [
      makeM("m1", "Milestone 1", 1, "pending"),
      makeM("m2", "Milestone 2", 2, "passed"),
    ];
    renderWithI18n(<FeatureMilestonesSection featureId="f1" />);
    expect(screen.getByText("Milestone 1")).toBeInTheDocument();
    expect(screen.getByText("Milestone 2")).toBeInTheDocument();
  });

  it("shows a testid row per milestone", () => {
    mockMilestones.value = [makeM("m1", "Phase 1", 1, "pending")];
    renderWithI18n(<FeatureMilestonesSection featureId="f1" />);
    expect(screen.getByTestId("milestone-row-m1")).toBeInTheDocument();
  });

  it("renders validation status for each milestone", () => {
    mockMilestones.value = [makeM("m1", "Phase 1", 1, "passed")];
    renderWithI18n(<FeatureMilestonesSection featureId="f1" />);
    expect(screen.getByTestId("validation-status-m1")).toHaveTextContent("passed");
  });

  it("renders failed validation status", () => {
    mockMilestones.value = [makeM("m1", "Phase 1", 1, "failed")];
    renderWithI18n(<FeatureMilestonesSection featureId="f1" />);
    expect(screen.getByTestId("validation-status-m1")).toHaveTextContent("failed");
  });

  it("renders pending validation status", () => {
    mockMilestones.value = [makeM("m1", "Phase 1", 1, "pending")];
    renderWithI18n(<FeatureMilestonesSection featureId="f1" />);
    expect(screen.getByTestId("validation-status-m1")).toHaveTextContent("pending");
  });

  it("renders DoD assertions under a milestone", () => {
    mockMilestones.value = [makeM("m1", "Phase 1", 1, "failed")];
    mockDodAssertions.value = {
      m1: [
        makeAssertion("a1", "m1", "Tests pass", "passed"),
        makeAssertion("a2", "m1", "No regressions", "failed"),
      ],
    };
    renderWithI18n(<FeatureMilestonesSection featureId="f1" />);
    expect(screen.getByText("Tests pass")).toBeInTheDocument();
    expect(screen.getByText("No regressions")).toBeInTheDocument();
    mockDodAssertions.value = {};
  });

  it("renders assertion testid for each DoD item", () => {
    mockMilestones.value = [makeM("m1", "Phase 1", 1, "pending")];
    mockDodAssertions.value = {
      m1: [makeAssertion("a1", "m1", "All tests green", "pending")],
    };
    renderWithI18n(<FeatureMilestonesSection featureId="f1" />);
    expect(screen.getByTestId("dod-assertion-a1")).toBeInTheDocument();
    mockDodAssertions.value = {};
  });

  it("shows multiple milestones in order", () => {
    mockMilestones.value = [
      makeM("m1", "First Milestone", 1, "passed"),
      makeM("m2", "Second Milestone", 2, "pending"),
      makeM("m3", "Third Milestone", 3, "pending"),
    ];
    renderWithI18n(<FeatureMilestonesSection featureId="f1" />);
    const row1 = screen.getByTestId("milestone-row-m1");
    const row2 = screen.getByTestId("milestone-row-m2");
    const row3 = screen.getByTestId("milestone-row-m3");
    expect(row1.compareDocumentPosition(row2)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
    expect(row2.compareDocumentPosition(row3)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
  });
});
