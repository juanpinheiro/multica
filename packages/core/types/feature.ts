import type { IssueStatus, IssuePriority } from "./issue";

export type FeatureStatus = "planned" | "in_progress" | "paused" | "completed" | "cancelled";

export type FeaturePriority = "urgent" | "high" | "medium" | "low" | "none";

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
