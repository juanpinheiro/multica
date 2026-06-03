import { describe, it, expect } from "vitest";
import { paths } from "./paths";

describe("paths.workspace(slug)", () => {
  const ws = paths.workspace("acme");

  it("builds workspace paths with slug prefix", () => {
    expect(ws.root()).toBe("/acme/live");
    expect(ws.live()).toBe("/acme/live");
    expect(ws.initiatives()).toBe("/acme/initiatives");
    expect(ws.initiativeDetail("init-1")).toBe("/acme/initiatives/init-1");
    expect(ws.initiativeIssue("init-1", "issue-2")).toBe(
      "/acme/initiatives/init-1/issues/issue-2",
    );
    expect(ws.decisions()).toBe("/acme/decisions");
    expect(ws.usage()).toBe("/acme/usage");
    expect(ws.issues()).toBe("/acme/issues");
    expect(ws.issueDetail("abc-123")).toBe("/acme/issues/abc-123");
    expect(ws.autopilots()).toBe("/acme/autopilots");
    expect(ws.autopilotDetail("a1")).toBe("/acme/autopilots/a1");
    expect(ws.agents()).toBe("/acme/agents");
    expect(ws.inbox()).toBe("/acme/inbox");
    expect(ws.myIssues()).toBe("/acme/my-issues");
    expect(ws.runtimes()).toBe("/acme/runtimes");
    expect(ws.skills()).toBe("/acme/skills");
    expect(ws.skillDetail("skl_123")).toBe("/acme/skills/skl_123");
    expect(ws.settings()).toBe("/acme/settings");
    expect(ws.attachmentPreview("att_42")).toBe("/acme/attachments/att_42/preview");
  });

  it("URL-encodes special characters in ids", () => {
    expect(ws.issueDetail("id with space")).toBe("/acme/issues/id%20with%20space");
  });
});

describe("paths (global)", () => {
  it("root returns /", () => {
    expect(paths.root()).toBe("/");
  });
});
