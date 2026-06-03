"use client";

import { useRouter, usePathname, useSearchParams } from "next/navigation";
import { useEffect } from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";

export const VARIANTS = [
  { key: "A", name: "Board (Linear-like)" },
  { key: "B", name: "Activity feed" },
  { key: "C", name: "Initiative tiles" },
] as const;

export function PrototypeSwitcher({ current }: { current: string }) {
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();

  const idx = Math.max(
    0,
    VARIANTS.findIndex((v) => v.key === current),
  );
  const cur = VARIANTS[idx] ?? VARIANTS[0];

  const setVariant = (key: string) => {
    const sp = new URLSearchParams(params?.toString());
    sp.set("variant", key);
    router.replace(`${pathname}?${sp.toString()}`);
  };

  const cycle = (dir: 1 | -1) => {
    const next = VARIANTS[(idx + dir + VARIANTS.length) % VARIANTS.length];
    if (next) setVariant(next.key);
  };

  useEffect(() => {
    const isEditable = (el: EventTarget | null): boolean => {
      if (!(el instanceof HTMLElement)) return false;
      const tag = el.tagName.toLowerCase();
      return (
        tag === "input" ||
        tag === "textarea" ||
        el.isContentEditable
      );
    };
    const onKey = (e: KeyboardEvent) => {
      if (isEditable(e.target)) return;
      if (e.key === "ArrowLeft") {
        e.preventDefault();
        cycle(-1);
      } else if (e.key === "ArrowRight") {
        e.preventDefault();
        cycle(1);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [idx]);

  if (process.env.NODE_ENV === "production") return null;

  return (
    <div className="fixed bottom-6 left-1/2 z-50 -translate-x-1/2">
      <div className="flex items-center gap-1 rounded-full border border-border bg-background/95 px-2 py-1 shadow-2xl backdrop-blur">
        <button
          aria-label="previous variant"
          onClick={() => cycle(-1)}
          className="rounded-full p-2 hover:bg-muted"
        >
          <ChevronLeft className="size-4" />
        </button>
        <div className="px-3 py-1 text-sm tabular-nums">
          <span className="font-semibold">{cur.key}</span>
          <span className="text-muted-foreground"> — {cur.name}</span>
        </div>
        <button
          aria-label="next variant"
          onClick={() => cycle(1)}
          className="rounded-full p-2 hover:bg-muted"
        >
          <ChevronRight className="size-4" />
        </button>
      </div>
      <div className="mt-2 text-center text-[10px] uppercase tracking-wider text-muted-foreground">
        prototype • ← / → to cycle
      </div>
    </div>
  );
}
