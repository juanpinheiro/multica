import type { ReactNode } from "react";
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { I18nProvider } from "@multica/core/i18n/react";
import enCommon from "../../locales/en/common.json";
import enSettings from "../../locales/en/settings.json";

const reposRef = vi.hoisted(() => ({
  current: [] as Array<{ id: string; name: string; remote_url: string }>,
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({ data: reposRef.current }),
}));

vi.mock("@multica/core/hooks", () => ({
  useWorkspaceId: () => "workspace-1",
}));

vi.mock("@multica/core/workspace/queries", () => ({
  repoListOptions: () => ({ queryKey: ["repos"], queryFn: vi.fn() }),
}));

import { RepositoriesTab } from "./repositories-tab";

const TEST_RESOURCES = {
  en: { common: enCommon, settings: enSettings },
};

function I18nWrapper({ children }: { children: ReactNode }) {
  return (
    <I18nProvider locale="en" resources={TEST_RESOURCES}>
      {children}
    </I18nProvider>
  );
}

describe("RepositoriesTab — read-only list", () => {
  it("renders the workspace repos with name and remote url", () => {
    reposRef.current = [
      { id: "r1", name: "backend", remote_url: "github.com/team/backend" },
      { id: "r2", name: "frontend", remote_url: "github.com/team/frontend" },
    ];
    render(<RepositoriesTab />, { wrapper: I18nWrapper });
    expect(screen.getByText("backend")).toBeTruthy();
    expect(screen.getByText("github.com/team/backend")).toBeTruthy();
    expect(screen.getByText("frontend")).toBeTruthy();
    expect(screen.getByText("github.com/team/frontend")).toBeTruthy();
  });

  it("shows the empty state when there are no repos", () => {
    reposRef.current = [];
    render(<RepositoriesTab />, { wrapper: I18nWrapper });
    expect(screen.getByText(enSettings.repositories.empty)).toBeTruthy();
  });
});
