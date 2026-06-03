import type { Workspace } from "../types";
import { paths } from "./paths";

/**
 * Determines where to send the user after authentication.
 *
 * Priority:
 *   workspace[0] → /<first.slug>/live
 *   no workspaces → / (renders NoWorkspacePage empty state)
 */
export function resolvePostAuthDestination(workspaces: Workspace[]): string {
  const first = workspaces[0];
  if (first) {
    return paths.workspace(first.slug).live();
  }
  return paths.root();
}
