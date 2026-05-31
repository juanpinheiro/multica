// @vitest-environment jsdom

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { Agent } from "@multica/core/types";
import { I18nProvider } from "@multica/core/i18n/react";
import enCommon from "../../../locales/en/common.json";
import enAgents from "../../../locales/en/agents.json";

const TEST_RESOURCES = { en: { common: enCommon, agents: enAgents } };

const mockUpdateAgent = vi.hoisted(() => vi.fn());

vi.mock("@multica/core/api", () => ({
  api: {
    updateAgent: (...args: unknown[]) => mockUpdateAgent(...args),
  },
}));

vi.mock("sonner", () => ({
  toast: { error: vi.fn(), success: vi.fn() },
}));

import { McpTab } from "./mcp-tab";

const BASE_AGENT: Agent = {
  id: "agent-1",
  workspace_id: "ws-1",
  runtime_id: "runtime-1",
  name: "Agent",
  description: "",
  instructions: "",
  avatar_url: null,
  runtime_mode: "local",
  runtime_config: {},
  custom_args: [],
  visibility: "workspace",
  status: "idle",
  max_concurrent_tasks: 1,
  model: "",
  owner_id: "user-1",
  skills: [],
  created_at: "2026-04-16T00:00:00Z",
  updated_at: "2026-04-16T00:00:00Z",
  archived_at: null,
  archived_by: null,
};

function renderMcpTab(agentOverrides: Partial<Agent> = {}, onDirtyChange?: (v: boolean) => void) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const agent = { ...BASE_AGENT, ...agentOverrides };
  return render(
    <I18nProvider locale="en" resources={TEST_RESOURCES}>
      <QueryClientProvider client={qc}>
        <McpTab
          agent={agent}
          onSave={vi.fn()}
          onDirtyChange={onDirtyChange}
        />
      </QueryClientProvider>
    </I18nProvider>,
  );
}

describe("McpTab", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUpdateAgent.mockResolvedValue({});
  });

  it("renders existing config in textarea", () => {
    const config = { mcpServers: { foo: { command: "npx" } } };
    renderMcpTab({ mcp_config: config });
    const textarea = screen.getByRole("textbox");
    expect(textarea).toHaveValue(JSON.stringify(config, null, 2));
  });

  it("shows redacted notice when mcp_config_redacted is true", () => {
    renderMcpTab({ mcp_config: null, mcp_config_redacted: true });
    expect(screen.getByText(/hidden/i)).toBeInTheDocument();
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
  });

  it("shows empty state message when mcp_config is null and not redacted", () => {
    renderMcpTab({ mcp_config: null });
    expect(screen.getByText(/No MCP configuration set/i)).toBeInTheDocument();
    expect(screen.getByRole("textbox")).toBeInTheDocument();
  });

  it("calls onDirtyChange(true) when textarea is edited", () => {
    const onDirtyChange = vi.fn();
    renderMcpTab({ mcp_config: null }, onDirtyChange);
    const textarea = screen.getByRole("textbox");
    fireEvent.change(textarea, { target: { value: '{"mcpServers":{}}' } });
    expect(onDirtyChange).toHaveBeenCalledWith(true);
  });

  it("shows invalid JSON error when textarea contains malformed JSON", () => {
    renderMcpTab({ mcp_config: null });
    const textarea = screen.getByRole("textbox");
    fireEvent.change(textarea, { target: { value: "not json" } });
    expect(screen.getByRole("alert")).toHaveTextContent(/Invalid JSON/i);
  });

  it("shows schema error when JSON lacks mcpServers key", () => {
    renderMcpTab({ mcp_config: null });
    const textarea = screen.getByRole("textbox");
    fireEvent.change(textarea, { target: { value: '{"other": 1}' } });
    expect(screen.getByRole("alert")).toHaveTextContent(/mcpServers/i);
  });

  it("calls onSave with parsed config on save", async () => {
    const onSave = vi.fn().mockResolvedValue(undefined);
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <I18nProvider locale="en" resources={TEST_RESOURCES}>
        <QueryClientProvider client={qc}>
          <McpTab
            agent={{ ...BASE_AGENT, mcp_config: null }}
            onSave={onSave}
          />
        </QueryClientProvider>
      </I18nProvider>,
    );
    const validConfig = '{"mcpServers":{"s":{"command":"npx"}}}';
    fireEvent.change(screen.getByRole("textbox"), { target: { value: validConfig } });
    fireEvent.click(screen.getByRole("button", { name: /save/i }));
    await waitFor(() =>
      expect(onSave).toHaveBeenCalledWith({ mcp_config: JSON.parse(validConfig) }),
    );
  });

  it("calls onSave with null when textarea is cleared", async () => {
    const onSave = vi.fn().mockResolvedValue(undefined);
    const config = { mcpServers: { foo: { command: "npx" } } };
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <I18nProvider locale="en" resources={TEST_RESOURCES}>
        <QueryClientProvider client={qc}>
          <McpTab
            agent={{ ...BASE_AGENT, mcp_config: config }}
            onSave={onSave}
          />
        </QueryClientProvider>
      </I18nProvider>,
    );
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "" } });
    fireEvent.click(screen.getByRole("button", { name: /save/i }));
    await waitFor(() =>
      expect(onSave).toHaveBeenCalledWith({ mcp_config: null }),
    );
  });
});
