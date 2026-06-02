import type { Label } from "./label";

export type IssueStatus =
  | "backlog"
  | "todo"
  | "in_progress"
  | "in_review"
  | "done"
  | "blocked"
  | "cancelled";

export type IssuePriority = "urgent" | "high" | "medium" | "low" | "none";

export type IssueAssigneeType = "agent";
export type IssueActorType = "member" | "agent";

/**
 * Per-issue metadata is a flat KV map agents use to record pipeline state
 * (PR number, pipeline_status, waiting_on, ...). Values are primitives only —
 * string / number / bool — enforced by both the API and the DB. Always
 * present in responses (empty object when unset) so reads don't need a
 * nil guard on the parent field.
 */
export type IssueMetadataValue = string | number | boolean;
export type IssueMetadata = Record<string, IssueMetadataValue>;

export interface Issue {
  id: string;
  workspace_id: string;
  number: number;
  identifier: string;
  title: string;
  description: string | null;
  status: IssueStatus;
  priority: IssuePriority;
  assignee_type: IssueAssigneeType | null;
  assignee_id: string | null;
  creator_type: IssueActorType;
  creator_id: string;
  parent_issue_id: string | null;
  feature_id: string | null;
  // The Milestone this Issue belongs to within its Initiative (ADR-0002).
  // Optional until the milestone table and its wiring land in issue 07.
  milestone_id?: string | null;
  position: number;
  start_date: string | null;
  due_date: string | null;
  metadata: IssueMetadata;
  labels?: Label[];
  created_at: string;
  updated_at: string;
}
