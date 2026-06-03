// Chrome (sidebar / top-nav) prototype. Three radically different shells
// over the same static body so the difference between them is the chrome
// — not the content. Throwaway, delete with the prototype once a shape
// is chosen.

import { ChromeRail } from "./_chrome-a-rail";
import { ChromeWide } from "./_chrome-b-wide";
import { ChromeTopNav } from "./_chrome-c-topnav";
import { ChromeSwitcher } from "./_chrome-switcher";

export const dynamic = "force-dynamic";

export default async function ChromePrototypePage({
  searchParams,
}: {
  searchParams: Promise<{ chrome?: string }>;
}) {
  const sp = await searchParams;
  const chrome = (sp.chrome ?? "A").toUpperCase();

  return (
    <>
      {chrome === "A" && <ChromeRail />}
      {chrome === "B" && <ChromeWide />}
      {chrome === "C" && <ChromeTopNav />}
      {!["A", "B", "C"].includes(chrome) && <ChromeRail />}
      <ChromeSwitcher current={chrome} />
    </>
  );
}
