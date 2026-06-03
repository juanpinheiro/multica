import { describe, it, expect } from "vitest";
import { paths } from "./paths";

// C4 — link-handler's WORKSPACE_ROUTE_SEGMENTS must match paths.workspace's
// parameterless method names. We can't import WORKSPACE_ROUTE_SEGMENTS here
// because link-handler is in packages/views (no inverse import allowed), so
// we hardcode the expected list and assert paths.workspace produces the same
// keys. If you change either, BOTH need to be updated — the test catches drift.
describe("paths.workspace() shape", () => {
  it("exposes the expected parameterless workspace route methods", () => {
    const ws = paths.workspace("__probe__");
    const parameterlessRoutes = Object.entries(ws)
      .filter(([, fn]) => typeof fn === "function" && fn.length === 0)
      .map(([key]) => key);

    expect(new Set(parameterlessRoutes)).toEqual(
      new Set([
        "root",
        "live",
        "initiatives",
        "decisions",
        "usage",
        "issues",
        "autopilots",
        "agents",
        "inbox",
        "myIssues",
        "runtimes",
        "skills",
        "settings",
      ]),
    );
  });

  it("each parameterless route emits /{slug}/{segment}", () => {
    const ws = paths.workspace("acme");
    const expectedSegments: Array<[string, string]> = [
      ["live", "live"],
      ["initiatives", "initiatives"],
      ["decisions", "decisions"],
      ["usage", "usage"],
      ["issues", "issues"],
      ["autopilots", "autopilots"],
      ["agents", "agents"],
      ["inbox", "inbox"],
      ["myIssues", "my-issues"],
      ["runtimes", "runtimes"],
      ["skills", "skills"],
      ["settings", "settings"],
    ];
    const wsAsAny = ws as unknown as Record<string, () => string>;
    for (const [method, segment] of expectedSegments) {
      const fn = wsAsAny[method];
      expect(typeof fn).toBe("function");
      expect(fn!()).toBe(`/acme/${segment}`);
    }
  });
});
