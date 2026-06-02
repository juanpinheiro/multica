import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import type { Liveness, ActivityCounters } from "@multica/core/tasks";

import { BoardCardLiveLayer } from "./board-card";

function makeLiveness(overrides: Partial<Liveness> = {}): Liveness {
  return {
    active: false,
    phase: "claim",
    heartbeat: "fresh",
    quietMs: 0,
    elapsedMs: 0,
    waiting: null,
    ...overrides,
  };
}

function makeCounters(overrides: Partial<ActivityCounters> = {}): ActivityCounters {
  return { activityCount: 0, elapsedMs: 0, ...overrides };
}

describe("BoardCardLiveLayer", () => {
  it("renders shimmer when liveness is active", () => {
    render(<BoardCardLiveLayer liveness={makeLiveness({ active: true, phase: "run" })} />);
    expect(screen.getByTestId("task-progress-shimmer")).toBeInTheDocument();
  });

  it("renders nothing when liveness is null", () => {
    const { container } = render(<BoardCardLiveLayer liveness={null} />);
    expect(screen.queryByTestId("task-progress-shimmer")).not.toBeInTheDocument();
    expect(container.firstChild).toBeNull();
  });

  it("renders nothing when liveness is inactive", () => {
    render(<BoardCardLiveLayer liveness={makeLiveness({ active: false })} />);
    expect(screen.queryByTestId("task-progress-shimmer")).not.toBeInTheDocument();
  });

  it("renders shimmer for waiting phase (active: true)", () => {
    render(<BoardCardLiveLayer liveness={makeLiveness({ active: true, phase: "claim" })} />);
    expect(screen.getByTestId("task-progress-shimmer")).toBeInTheDocument();
  });

  describe("heartbeat", () => {
    it("renders a fresh heartbeat as 'now'", () => {
      render(
        <BoardCardLiveLayer
          liveness={makeLiveness({ active: true, phase: "run", heartbeat: "fresh", quietMs: 1_500 })}
        />,
      );
      const heartbeat = screen.getByTestId("heartbeat");
      expect(heartbeat).toHaveTextContent("now");
      expect(heartbeat).not.toHaveTextContent("quiet");
    });

    it("renders a quiet heartbeat as 'quiet Ns'", () => {
      render(
        <BoardCardLiveLayer
          liveness={makeLiveness({ active: true, phase: "run", heartbeat: "quiet", quietMs: 24_000 })}
        />,
      );
      expect(screen.getByTestId("heartbeat")).toHaveTextContent("quiet 24s");
    });

    it("renders no heartbeat when inactive", () => {
      render(<BoardCardLiveLayer liveness={makeLiveness({ active: false })} />);
      expect(screen.queryByTestId("heartbeat")).not.toBeInTheDocument();
    });
  });

  describe("phase stepper", () => {
    it("marks run step as current for run phase", () => {
      render(<BoardCardLiveLayer liveness={makeLiveness({ active: true, phase: "run" })} />);
      expect(screen.getByTestId("phase-step-run")).toHaveAttribute("aria-current", "step");
    });

    it("marks claim step as current for claim phase", () => {
      render(<BoardCardLiveLayer liveness={makeLiveness({ active: true, phase: "claim" })} />);
      expect(screen.getByTestId("phase-step-claim")).toHaveAttribute("aria-current", "step");
    });

    it("marks pr step as current for pr phase", () => {
      render(<BoardCardLiveLayer liveness={makeLiveness({ active: true, phase: "pr" })} />);
      expect(screen.getByTestId("phase-step-pr")).toHaveAttribute("aria-current", "step");
    });

    it("does not mark prior steps as current for run phase", () => {
      render(<BoardCardLiveLayer liveness={makeLiveness({ active: true, phase: "run" })} />);
      expect(screen.getByTestId("phase-step-claim")).not.toHaveAttribute("aria-current");
      expect(screen.getByTestId("phase-step-push")).not.toHaveAttribute("aria-current");
      expect(screen.getByTestId("phase-step-pr")).not.toHaveAttribute("aria-current");
    });

    it("renders all four steps", () => {
      render(<BoardCardLiveLayer liveness={makeLiveness({ active: true, phase: "run" })} />);
      expect(screen.getByTestId("phase-step-claim")).toBeInTheDocument();
      expect(screen.getByTestId("phase-step-run")).toBeInTheDocument();
      expect(screen.getByTestId("phase-step-push")).toBeInTheDocument();
      expect(screen.getByTestId("phase-step-pr")).toBeInTheDocument();
    });
  });

  describe("waiting block", () => {
    it("renders waiting block when liveness.waiting is set", () => {
      render(
        <BoardCardLiveLayer
          liveness={makeLiveness({
            active: true,
            phase: "claim",
            waiting: { reason: "umbrella directory /code is in use by task abc-123", holderKey: null },
          })}
        />,
      );
      expect(screen.getByTestId("waiting-block")).toBeInTheDocument();
    });

    it("shows the wait reason in the waiting block", () => {
      const reason = "umbrella directory /code/project is in use by task abc-123";
      render(
        <BoardCardLiveLayer
          liveness={makeLiveness({ active: true, phase: "claim", waiting: { reason, holderKey: null } })}
        />,
      );
      expect(screen.getByTestId("waiting-block")).toHaveTextContent(reason);
    });

    it("shows the holderKey in the waiting block when present", () => {
      render(
        <BoardCardLiveLayer
          liveness={makeLiveness({
            active: true,
            phase: "claim",
            waiting: { reason: "some reason", holderKey: "MUL-42" },
          })}
        />,
      );
      expect(screen.getByTestId("waiting-block")).toHaveTextContent("MUL-42");
    });

    it("renders without crashing when holderKey is null", () => {
      render(
        <BoardCardLiveLayer
          liveness={makeLiveness({
            active: true,
            phase: "claim",
            waiting: { reason: "some reason", holderKey: null },
          })}
        />,
      );
      const block = screen.getByTestId("waiting-block");
      expect(block).toBeInTheDocument();
      expect(block).toHaveTextContent("some reason");
    });

    it("does not render waiting block when liveness.waiting is null", () => {
      render(
        <BoardCardLiveLayer liveness={makeLiveness({ active: true, phase: "run", waiting: null })} />,
      );
      expect(screen.queryByTestId("waiting-block")).not.toBeInTheDocument();
    });

    it("does not render waiting block when liveness is inactive", () => {
      render(
        <BoardCardLiveLayer
          liveness={makeLiveness({ active: false, waiting: { reason: "some reason", holderKey: null } })}
        />,
      );
      expect(screen.queryByTestId("waiting-block")).not.toBeInTheDocument();
    });
  });

  describe("activity counters", () => {
    it("renders elapsed time and action count when counters are provided", () => {
      render(
        <BoardCardLiveLayer
          liveness={makeLiveness({ active: true, phase: "run" })}
          counters={makeCounters({ activityCount: 3, elapsedMs: 90_000 })}
        />,
      );
      const el = screen.getByTestId("activity-counters");
      expect(el).toHaveTextContent("1m");
      expect(el).toHaveTextContent("3 actions");
    });

    it("renders singular 'action' for count of 1", () => {
      render(
        <BoardCardLiveLayer
          liveness={makeLiveness({ active: true, phase: "run" })}
          counters={makeCounters({ activityCount: 1, elapsedMs: 5_000 })}
        />,
      );
      expect(screen.getByTestId("activity-counters")).toHaveTextContent("1 action");
    });

    it("does not render action count when activityCount is 0", () => {
      render(
        <BoardCardLiveLayer
          liveness={makeLiveness({ active: true, phase: "run" })}
          counters={makeCounters({ activityCount: 0, elapsedMs: 10_000 })}
        />,
      );
      const el = screen.getByTestId("activity-counters");
      expect(el).not.toHaveTextContent("action");
    });

    it("renders nothing when counters are null", () => {
      render(
        <BoardCardLiveLayer
          liveness={makeLiveness({ active: true, phase: "run" })}
          counters={null}
        />,
      );
      expect(screen.queryByTestId("activity-counters")).not.toBeInTheDocument();
    });

    it("renders nothing when liveness is inactive even with counters", () => {
      render(
        <BoardCardLiveLayer
          liveness={makeLiveness({ active: false })}
          counters={makeCounters({ activityCount: 5, elapsedMs: 60_000 })}
        />,
      );
      expect(screen.queryByTestId("activity-counters")).not.toBeInTheDocument();
    });
  });
});
