import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { paths } from "@multica/core/paths";

export default async function RootPage() {
  const lastSlug = (await cookies()).get("last_workspace_slug")?.value;
  if (lastSlug) {
    redirect(paths.workspace(lastSlug).issues());
  }
  redirect(paths.newWorkspace());
}
