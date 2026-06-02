import type { IssueStatus, IssuePriority } from "./issue";
import type { InitiativeStatus } from "../initiative/status";

// A Feature is the persisted shape of an Initiative (ADR-0002); its status is
// the Initiative lifecycle state. The `FeatureStatus` alias is kept while the
// `feature` table name and API surface remain.
export type FeatureStatus = InitiativeStatus;

export type FeaturePriority = "urgent" | "high" | "medium" | "low" | "none";

// Mode is the planning-time autonomy choice for an Initiative (ADR-0005): AFK is
// one big autonomous PR, HITL is several reviewed PRs.
export type InitiativeMode = "hitl" | "afk";

export interface Feature {
  id: string;
  workspace_id: string;
  title: string;
  description: string | null;
  icon: string | null;
  status: FeatureStatus;
  priority: FeaturePriority;
  lead_type: "member" | "agent" | null;
  lead_id: string | null;
  branch_slug: string | null;
  // Mode + the budget/failure-tolerance fields feed the Tripwire/Budget safety
  // net (ADR-0005). A zero budget means "no cap" for that dimension.
  mode: InitiativeMode;
  budget_tokens: number;
  budget_runs: number;
  budget_seconds: number;
  failure_tolerance: number;
  created_at: string;
  updated_at: string;
  issue_count: number;
  done_count: number;
  resource_count: number;
}

export interface CreateFeatureRequest {
  title: string;
  description?: string;
  icon?: string;
  status?: FeatureStatus;
  priority?: FeaturePriority;
  lead_type?: "member" | "agent";
  lead_id?: string;
  // Resources to attach in the same transaction as the feature. Server returns
  // 4xx (and rolls back) if any one is invalid or duplicate.
  resources?: CreateFeatureResourceRequest[];
}

export interface UpdateFeatureRequest {
  title?: string;
  description?: string | null;
  icon?: string | null;
  status?: FeatureStatus;
  priority?: FeaturePriority;
  lead_type?: "member" | "agent" | null;
  lead_id?: string | null;
}

export interface ListFeaturesResponse {
  features: Feature[];
  total: number;
}

// FeatureResource is a typed pointer from a feature to an external resource.
// The resource_ref shape depends on resource_type (e.g. github_repo carries
// { url, default_branch_hint? }). New types add a case in
// validateAndNormalizeResourceRef on the server and a renderer in the UI;
// no schema or type changes required.
export type FeatureResourceType = "github_repo";

export interface GithubRepoResourceRef {
  url: string;
  default_branch_hint?: string;
}

export interface FeatureResource {
  id: string;
  feature_id: string;
  workspace_id: string;
  resource_type: FeatureResourceType;
  resource_ref: GithubRepoResourceRef | Record<string, unknown>;
  label: string | null;
  position: number;
  created_at: string;
  created_by: string | null;
}

export interface CreateFeatureResourceRequest {
  resource_type: FeatureResourceType;
  resource_ref: GithubRepoResourceRef | Record<string, unknown>;
  label?: string;
  position?: number;
}

export interface ListFeatureResourcesResponse {
  resources: FeatureResource[];
  total: number;
}

export interface FeatureIssueSummary {
  id: string;
  identifier: string;
  title: string;
  status: IssueStatus;
  priority: IssuePriority;
  repo_id?: string | null;
  repo_name?: string | null;
}

export interface FeatureBlockedIssueSummary extends FeatureIssueSummary {
  blocked_by: string[];
}

export interface FeaturePRSummary {
  number: number;
  html_url: string;
  state: string;
  title: string;
  repo_id?: string | null;
}

export interface FeatureIssuesResponse {
  ready_now: FeatureIssueSummary[];
  blocked: FeatureBlockedIssueSummary[];
  pull_requests: FeaturePRSummary[];
}
