import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";
import { issueKeys } from "../issues/queries";
import { workspaceKeys } from "../workspace/queries";
import type { Workspace } from "../types";
import { applyWorkspaceUpdatedToCache } from "./use-realtime-sync";

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });
}

describe("applyWorkspaceUpdatedToCache", () => {
  const wsId = "ws-1";

  function workspace(overrides: Partial<Workspace> = {}): Workspace {
    return {
      id: wsId,
      name: "Test",
      slug: "test",
      description: null,
      context: null,
      settings: {},
      issue_prefix: "TES",
      created_at: "2026-05-18T00:00:00Z",
      updated_at: "2026-05-18T00:00:00Z",
      ...overrides,
    };
  }

  it("invalidates issue cache when issue_prefix changes", () => {
    const qc = createQueryClient();
    qc.setQueryData<Workspace[]>(workspaceKeys.list(), [
      workspace({ issue_prefix: "TES" }),
    ]);
    const invalidate = vi.spyOn(qc, "invalidateQueries");

    applyWorkspaceUpdatedToCache(qc, {
      workspace: workspace({ issue_prefix: "NEW" }),
    });

    expect(invalidate).toHaveBeenCalledWith({
      queryKey: issueKeys.all(wsId),
    });
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: workspaceKeys.list(),
    });
  });

  it("does not invalidate issue cache when only non-prefix fields change", () => {
    const qc = createQueryClient();
    qc.setQueryData<Workspace[]>(workspaceKeys.list(), [
      workspace({ issue_prefix: "TES", name: "Old name" }),
    ]);
    const invalidate = vi.spyOn(qc, "invalidateQueries");

    applyWorkspaceUpdatedToCache(qc, {
      workspace: workspace({ issue_prefix: "TES", name: "New name" }),
    });

    expect(invalidate).not.toHaveBeenCalledWith({
      queryKey: issueKeys.all(wsId),
    });
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: workspaceKeys.list(),
    });
  });

  it("invalidates issue cache when the workspace isn't in the cached list yet", () => {
    const qc = createQueryClient();
    const invalidate = vi.spyOn(qc, "invalidateQueries");

    applyWorkspaceUpdatedToCache(qc, {
      workspace: workspace({ issue_prefix: "NEW" }),
    });

    expect(invalidate).toHaveBeenCalledWith({
      queryKey: issueKeys.all(wsId),
    });
  });
});
