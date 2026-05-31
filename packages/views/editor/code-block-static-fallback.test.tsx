import { render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

// Simulate the real-world bug: lowlight.highlightAuto returns an empty tree
// (children: []) for some inputs, causing toHtml to return "". Before the fix
// the code element would be completely empty — the code vanished.
vi.mock("hast-util-to-html", () => ({
  toHtml: vi.fn(() => ""),
}));

import { CodeBlockStatic } from "./code-block-static";

describe("CodeBlockStatic — empty highlight fallback", () => {
  it("shows code as escaped text when the highlight tree produces empty html", () => {
    const { container } = render(
      <CodeBlockStatic language={undefined} body="<script>alert(1)</script>" />,
    );
    const code = container.querySelector("code");
    expect(code).not.toBeNull();
    // Content must be visible — not vanished when toHtml returns ""
    expect(code?.textContent).toBe("<script>alert(1)</script>");
    // Content must be escaped — no raw HTML injection
    expect(code?.innerHTML).toBe("&lt;script&gt;alert(1)&lt;/script&gt;");
  });

  it("shows code with a specified language as escaped text when the highlight tree produces empty html", () => {
    const { container } = render(
      <CodeBlockStatic language="typescript" body="const x = 1;" />,
    );
    const code = container.querySelector("code");
    expect(code?.textContent).toBe("const x = 1;");
  });
});
