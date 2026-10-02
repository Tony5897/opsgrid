import { PRIORITY_META, type Priority } from "@/features/work-orders/status";
import { cn } from "@/lib/utils";

interface PriorityIndicatorProps {
  priority: Priority;
  showLabel?: boolean;
  className?: string;
}

/** Signal-strength bars + label; the label is always available to assistive tech. */
export function PriorityIndicator({
  priority,
  showLabel = true,
  className,
}: PriorityIndicatorProps) {
  const meta = PRIORITY_META[priority];
  return (
    <span
      className={cn("inline-flex items-center gap-1.5 text-xs font-medium", className)}
      style={{ color: `var(--priority-${priority})` }}
    >
      <span aria-hidden className="flex h-3 items-end gap-px">
        {[1, 2, 3, 4].map((n) => (
          <span
            key={n}
            className="w-[3px] rounded-[1px]"
            style={{
              height: `${n * 25}%`,
              backgroundColor: n <= meta.bars ? "currentColor" : "var(--border-default)",
            }}
          />
        ))}
      </span>
      {showLabel ? meta.label : <span className="sr-only">{meta.label} priority</span>}
    </span>
  );
}
