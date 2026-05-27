import type { Workspace } from "../types";
import { paths } from "./paths";

/**
 * Determines where to send the user after authentication.
 *
 * Priority:
 *   workspace[0] → /<first.slug>/issues
 *   no workspaces → /workspaces/new
 */
export function resolvePostAuthDestination(workspaces: Workspace[]): string {
  const first = workspaces[0];
  if (first) {
    return paths.workspace(first.slug).issues();
  }
  return paths.newWorkspace();
}
