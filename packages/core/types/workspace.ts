export type MemberRole = "owner" | "admin" | "member";

export interface WorkspaceRepo {
  url: string;
  description?: string;
}

export interface Workspace {
  id: string;
  name: string;
  slug: string;
  description: string | null;
  context: string | null;
  settings: Record<string, unknown>;
  repos: WorkspaceRepo[];
  issue_prefix: string;
  created_at: string;
  updated_at: string;
}

export interface Member {
  id: string;
  workspace_id: string;
  user_id: string;
  role: MemberRole;
  created_at: string;
}

export interface User {
  id: string;
  name: string;
  email: string;
  avatar_url: string | null;
  /** Preferred UI language. null means "follow client/system". */
  language: string | null;
  /**
   * Free-form self-description (role, stack, preferences). Injected into
   * the agent brief so coding agents have cheap, durable context about
   * who is requesting the work. Server always returns a string —
   * NOT NULL DEFAULT '' at the column level, empty when unset.
   */
  profile_description: string;
  /** Pinned IANA tz; null means "use browser-detected tz at render time". */
  timezone: string | null;
  created_at: string;
  updated_at: string;
}

export interface MemberWithUser {
  id: string;
  workspace_id: string;
  user_id: string;
  role: MemberRole;
  created_at: string;
  name: string;
  email: string;
  avatar_url: string | null;
}
