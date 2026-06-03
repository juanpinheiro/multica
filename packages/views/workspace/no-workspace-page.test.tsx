import type { ReactNode } from "react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { I18nProvider } from "@multica/core/i18n/react";
import enCommon from "../locales/en/common.json";
import enWorkspace from "../locales/en/workspace.json";
import { NoWorkspacePage } from "./no-workspace-page";

const TEST_RESOURCES = {
  en: { common: enCommon, workspace: enWorkspace },
};

const navigate = vi.fn();

const workspaceListResult = vi.hoisted(() => ({
  current: [] as Array<{ id: string; slug: string }>,
}));

vi.mock("../navigation", () => ({
  useNavigation: () => ({ push: navigate, replace: navigate }),
}));

vi.mock("@multica/core/auth", () => {
  const useAuthStore = Object.assign(
    (sel?: (s: { user: { id: string } | null }) => unknown) =>
      sel ? sel({ user: { id: "user-1" } }) : { user: { id: "user-1" } },
    { getState: () => ({ user: { id: "user-1" } }) },
  );
  return { useAuthStore };
});

vi.mock("@multica/core/workspace/queries", () => ({
  workspaceListOptions: () => ({
    queryKey: ["workspaces"],
    queryFn: () => Promise.resolve(workspaceListResult.current),
  }),
}));

function I18nWrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient();
  return (
    <QueryClientProvider client={qc}>
      <I18nProvider locale="en" resources={TEST_RESOURCES}>
        {children}
      </I18nProvider>
    </QueryClientProvider>
  );
}

describe("NoWorkspacePage", () => {
  beforeEach(() => {
    navigate.mockReset();
    workspaceListResult.current = [];
  });

  it("renders the /setup-multica command in a code block", () => {
    render(<NoWorkspacePage />, { wrapper: I18nWrapper });
    expect(screen.getByText("/setup-multica")).toBeInTheDocument();
    expect(screen.getByText(/no workspace discovered/i)).toBeInTheDocument();
  });

  it("links to the standalone-install docs", () => {
    render(<NoWorkspacePage />, { wrapper: I18nWrapper });
    const link = screen.getByRole("link", { name: /standalone-install/i });
    expect(link).toHaveAttribute(
      "href",
      "https://github.com/multica-ai/multica/blob/main/docs/adr/0008-standalone-personal-install.md",
    );
  });

  it("redirects to the first workspace's live page when one exists", async () => {
    workspaceListResult.current = [{ id: "ws-1", slug: "acme" }];
    render(<NoWorkspacePage />, { wrapper: I18nWrapper });
    await waitFor(() => {
      expect(navigate).toHaveBeenCalledWith("/acme/live");
    });
  });
});
