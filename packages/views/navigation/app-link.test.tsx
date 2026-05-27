import { describe, it, expect, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { AppLink } from "./app-link";
import { NavigationProvider } from "./context";

const { push, replace, back, prefetch } = vi.hoisted(() => ({
  push: vi.fn(),
  replace: vi.fn(),
  back: vi.fn(),
  prefetch: vi.fn(),
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push, replace, back, prefetch }),
  usePathname: () => "/",
  useSearchParams: () => new URLSearchParams(),
}));

function renderLink(
  props: React.ComponentProps<typeof AppLink> = { href: "/issues" },
) {
  return render(
    <NavigationProvider>
      <AppLink {...props}>go</AppLink>
    </NavigationProvider>,
  );
}

describe("AppLink", () => {
  it("calls caller onClick BEFORE push so synchronous side effects (close menu, etc) commit before the transition starts", () => {
    push.mockClear();
    const order: string[] = [];
    push.mockImplementation(() => order.push("push"));
    renderLink({
      href: "/issues",
      onClick: () => order.push("onClick"),
    });

    fireEvent.click(screen.getByText("go"));
    expect(order).toEqual(["onClick", "push"]);
  });

  it("calls router.prefetch on hover, alongside the caller's onMouseEnter — neither is overridden by {...props}", () => {
    prefetch.mockClear();
    const callerMouseEnter = vi.fn();

    renderLink({
      href: "/issues",
      onMouseEnter: callerMouseEnter,
    });

    fireEvent.mouseEnter(screen.getByText("go"));
    expect(prefetch).toHaveBeenCalledWith("/issues");
    expect(callerMouseEnter).toHaveBeenCalledTimes(1);
  });

  it("calls router.prefetch on focus, alongside the caller's onFocus", () => {
    prefetch.mockClear();
    const callerFocus = vi.fn();

    renderLink({
      href: "/issues",
      onFocus: callerFocus,
    });

    fireEvent.focus(screen.getByText("go"));
    expect(prefetch).toHaveBeenCalledWith("/issues");
    expect(callerFocus).toHaveBeenCalledTimes(1);
  });

  it("modifier-click (cmd / ctrl) lets the browser handle the navigation natively and does NOT push", () => {
    push.mockClear();
    push.mockImplementation(() => {});
    renderLink();
    fireEvent.click(screen.getByText("go"), { metaKey: true });
    expect(push).not.toHaveBeenCalled();
  });

  it("a caller-supplied onClick passed via spread cannot silently override the navigation handler", () => {
    push.mockClear();
    push.mockImplementation(() => {});
    const spreadOnClick = vi.fn((e: React.MouseEvent) => e.preventDefault());

    render(
      <NavigationProvider>
        {/* simulate a caller that passes onClick through a spread bag */}
        <AppLink href="/issues" {...{ onClick: spreadOnClick }}>
          go
        </AppLink>
      </NavigationProvider>,
    );

    fireEvent.click(screen.getByText("go"));
    expect(spreadOnClick).toHaveBeenCalled();
    expect(push).toHaveBeenCalledWith("/issues");
  });
});
