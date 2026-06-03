import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { paths } from "@multica/core/paths";
import { NoWorkspacePage } from "@multica/views/workspace/no-workspace-page";

export default async function RootPage() {
  const lastSlug = (await cookies()).get("last_workspace_slug")?.value;
  if (lastSlug) {
    redirect(paths.workspace(lastSlug).live());
  }
  return <NoWorkspacePage />;
}
