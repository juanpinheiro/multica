// Initiative-runner landing prototype. Three radically different variants on
// the same route, switchable via ?variant=A|B|C. Throwaway.

import { VariantBoard } from "./_variant-a-board";
import { VariantFeed } from "./_variant-b-feed";
import { VariantCards } from "./_variant-c-cards";
import { PrototypeSwitcher } from "./_switcher";

export const dynamic = "force-dynamic";

export default async function InitiativeRunnerPrototypePage({
  searchParams,
}: {
  searchParams: Promise<{ variant?: string }>;
}) {
  const sp = await searchParams;
  const variant = (sp.variant ?? "A").toUpperCase();

  return (
    <>
      {variant === "A" && <VariantBoard variant="A" />}
      {variant === "B" && <VariantFeed variant="B" />}
      {variant === "C" && <VariantCards variant="C" />}
      {!["A", "B", "C"].includes(variant) && <VariantBoard variant="A" />}
      <PrototypeSwitcher current={variant} />
    </>
  );
}
