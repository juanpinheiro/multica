import { describe, it, expect } from "vitest";
import {
  INITIATIVE_STATUSES,
  canTransition,
  isInitiativeStatus,
  isTerminalStatus,
  type InitiativeStatus,
} from "./status";

describe("isInitiativeStatus", () => {
  it("accepts every known status", () => {
    for (const s of INITIATIVE_STATUSES) {
      expect(isInitiativeStatus(s)).toBe(true);
    }
  });

  it("rejects unknown or legacy values", () => {
    for (const s of ["", "planned", "in_progress", "completed", "DRAFT"]) {
      expect(isInitiativeStatus(s)).toBe(false);
    }
  });
});

describe("canTransition", () => {
  const legal: [InitiativeStatus, InitiativeStatus][] = [
    ["draft", "ready"],
    ["draft", "cancelled"],
    ["ready", "running"],
    ["ready", "blocked"],
    ["running", "in_review"],
    ["running", "done"],
    ["running", "blocked"],
    ["in_review", "done"],
    ["in_review", "running"],
    ["blocked", "ready"],
    ["blocked", "running"],
  ];

  const illegal: [InitiativeStatus, InitiativeStatus][] = [
    ["draft", "running"],
    ["draft", "done"],
    ["ready", "done"],
    ["ready", "in_review"],
    ["running", "ready"],
    ["running", "draft"],
    ["done", "running"],
    ["done", "ready"],
    ["cancelled", "ready"],
    ["draft", "draft"],
    ["running", "running"],
  ];

  it.each(legal)("allows %s → %s", (from, to) => {
    expect(canTransition(from, to)).toBe(true);
  });

  it.each(illegal)("rejects %s → %s", (from, to) => {
    expect(canTransition(from, to)).toBe(false);
  });
});

describe("isTerminalStatus", () => {
  it("marks done and cancelled terminal, nothing else", () => {
    expect(isTerminalStatus("done")).toBe(true);
    expect(isTerminalStatus("cancelled")).toBe(true);
    for (const s of ["draft", "ready", "running", "in_review", "blocked"] as const) {
      expect(isTerminalStatus(s)).toBe(false);
    }
  });
});
