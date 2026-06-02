// Pure status state machine for an Initiative — the PRD-level container that
// maps onto the backend `feature` table. This mirrors the canonical Go machine
// in server/internal/initiative/status.go; the two must stay in lockstep.
//
//   draft → ready → running → in_review → done
//
// with `blocked` as a tripwire/dependency pause and `cancelled` as an off-ramp.
// `done` and `cancelled` are terminal.

export type InitiativeStatus =
  | "draft"
  | "ready"
  | "running"
  | "in_review"
  | "done"
  | "blocked"
  | "cancelled";

export const INITIATIVE_STATUSES: InitiativeStatus[] = [
  "draft",
  "ready",
  "running",
  "in_review",
  "done",
  "blocked",
  "cancelled",
];

// Each status maps to the states it may transition into. A status absent from a
// list is an illegal target; terminal states have an empty list.
const ALLOWED: Record<InitiativeStatus, InitiativeStatus[]> = {
  draft: ["ready", "cancelled"],
  ready: ["running", "blocked", "cancelled"],
  running: ["in_review", "done", "blocked", "cancelled"],
  in_review: ["done", "running", "cancelled"],
  blocked: ["ready", "running", "cancelled"],
  done: [],
  cancelled: [],
};

export function isInitiativeStatus(value: unknown): value is InitiativeStatus {
  return (
    typeof value === "string" &&
    Object.prototype.hasOwnProperty.call(ALLOWED, value)
  );
}

// canTransition reports whether moving from → to is legal. Unknown statuses and
// self-transitions are never legal.
export function canTransition(from: InitiativeStatus, to: InitiativeStatus): boolean {
  return ALLOWED[from]?.includes(to) ?? false;
}

export function isTerminalStatus(status: InitiativeStatus): boolean {
  return ALLOWED[status]?.length === 0;
}
