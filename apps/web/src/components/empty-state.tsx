import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

interface EmptyStateProps {
  icon: LucideIcon;
  title: string;
  /** Teach the next action rather than just stating emptiness. */
  description: ReactNode;
  action?: ReactNode;
  className?: string;
}

export function EmptyState({ icon: Icon, title, description, action, className }: EmptyStateProps) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-border px-6 py-14 text-center",
        className,
      )}
    >
      <span className="flex size-11 items-center justify-center rounded-full bg-surface-2 text-text-secondary">
        <Icon aria-hidden className="size-5" />
      </span>
      <div className="max-w-md space-y-1">
        <h2 className="text-md font-semibold text-text-primary">{title}</h2>
        <p className="text-sm text-text-secondary">{description}</p>
      </div>
      {action}
    </div>
  );
}
