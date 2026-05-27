"use client";

import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useTransition,
} from "react";
import {
  usePathname,
  useRouter,
  useSearchParams,
} from "next/navigation";

interface Navigation {
  push(path: string): void;
  replace(path: string): void;
  back(): void;
  pathname: string;
  searchParams: URLSearchParams;
  getShareableUrl(path: string): string;
  prefetch(path: string): void;
}

type StartTransition = (cb: () => void) => void;

const NavigationPendingContext = createContext<boolean>(false);
const StartTransitionContext = createContext<StartTransition | null>(null);

export function NavigationProvider({ children }: { children: React.ReactNode }) {
  const [isPending, startTransition] = useTransition();
  return (
    <StartTransitionContext.Provider value={startTransition}>
      <NavigationPendingContext.Provider value={isPending}>
        {children}
      </NavigationPendingContext.Provider>
    </StartTransitionContext.Provider>
  );
}

export function useNavigation(): Navigation {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const startTransition = useContext(StartTransitionContext);
  if (!startTransition)
    throw new Error("useNavigation must be used within NavigationProvider");

  const push = useCallback(
    (path: string) => startTransition(() => router.push(path)),
    [router, startTransition],
  );
  const replace = useCallback(
    (path: string) => startTransition(() => router.replace(path)),
    [router, startTransition],
  );
  const back = useCallback(() => router.back(), [router]);
  const prefetch = useCallback((path: string) => router.prefetch(path), [router]);

  return useMemo<Navigation>(
    () => ({
      push,
      replace,
      back,
      pathname,
      searchParams: new URLSearchParams(searchParams.toString()),
      getShareableUrl: (path: string) =>
        typeof window === "undefined" ? path : window.location.origin + path,
      prefetch,
    }),
    [push, replace, back, pathname, searchParams, prefetch],
  );
}

/** True while a transition-wrapped push/replace is committing. */
export function useIsNavigating(): boolean {
  return useContext(NavigationPendingContext);
}
