// A Milestone is an ordered checkpoint within an Initiative (ADR-0002). Its
// validation status gates the next Milestone's Issues; the board resolves an
// Issue's milestone title from the workspace milestone list.
export type MilestoneValidationStatus = "pending" | "passed" | "failed";

export interface Milestone {
  id: string;
  workspace_id: string;
  feature_id: string;
  title: string;
  position: number;
  validation_status: MilestoneValidationStatus;
  created_at: string;
  updated_at: string;
}

export interface ListMilestonesResponse {
  milestones: Milestone[];
  total: number;
}
