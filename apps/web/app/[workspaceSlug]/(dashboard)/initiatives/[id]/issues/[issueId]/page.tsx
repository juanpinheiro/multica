"use client";

import { use } from "react";
import { IssueDetail } from "@multica/views/issues/components";
import { ErrorBoundary } from "@multica/ui/components/common/error-boundary";

export default function InitiativeIssueDetailPage({
  params,
}: {
  params: Promise<{ id: string; issueId: string }>;
}) {
  const { issueId } = use(params);
  return (
    <ErrorBoundary resetKeys={[issueId]}>
      <IssueDetail issueId={issueId} />
    </ErrorBoundary>
  );
}
