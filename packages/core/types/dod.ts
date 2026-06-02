// A Definition-of-Done assertion (ADR-0007): an initiative-level check tagged to
// a Milestone. `status` carries the latest validator verdict for the Milestone
// view; the per-Issue Acceptance Criteria view leaves it `pending`.
export type DodAssertionStatus = "pending" | "passed" | "failed";

export interface DodAssertion {
  id: string;
  workspace_id: string;
  feature_id: string;
  milestone_id: string;
  text: string;
  position: number;
  created_at: string;
  status: DodAssertionStatus;
  detail: string;
}

export interface ListDodAssertionsResponse {
  assertions: DodAssertion[];
}
