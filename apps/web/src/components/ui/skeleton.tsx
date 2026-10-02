import type { ComponentProps } from "react";
import { cn } from "@/lib/utils";

/** Layout-matching placeholder: pages load into skeletons, not spinners. */
export function Skeleton({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      aria-hidden
      className={cn("animate-pulse rounded-md bg-surface-3 motion-reduce:animate-none", className)}
      {...props}
    />
  );
}
