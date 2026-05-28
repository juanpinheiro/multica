"use client";

import { use } from "react";
import { FeatureDetail } from "@multica/views/features/components";

export default function FeatureDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  return <FeatureDetail featureId={id} />;
}
