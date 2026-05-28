import type { Feature } from "@multica/core/types";
import { cn } from "@multica/ui/lib/utils";

export type FeatureIconSize = "sm" | "md" | "lg";

export interface FeatureIconProps {
  feature?: Pick<Feature, "icon"> | null;
  size?: FeatureIconSize;
  className?: string;
}

const SIZE_CLASS: Record<FeatureIconSize, string> = {
  sm: "size-3.5 text-xs leading-none",
  md: "size-4 text-sm leading-none",
  lg: "size-6 text-2xl leading-none",
};

export function FeatureIcon({ feature, size = "sm", className }: FeatureIconProps) {
  return (
    <span
      aria-hidden="true"
      className={cn(
        "inline-flex shrink-0 items-center justify-center",
        SIZE_CLASS[size],
        className,
      )}
    >
      {feature?.icon || "📁"}
    </span>
  );
}
