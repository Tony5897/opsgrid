import { STATUS_META, type WorkOrderStatus } from "@/features/work-orders/status";
import { cn } from "@/lib/utils";

interface StatusBadgeProps {
  status: WorkOrderStatus;
  className?: string;
}

/** Color + icon + text: status is never conveyed by color alone (WCAG 1.4.1). */
export function StatusBadge({ status, className }: StatusBadgeProps) {
  const meta = STATUS_META[status];
  const Icon = meta.icon;
  return (
    <span
      data-status={status}
      title={meta.description}
      className={cn(
        "inline-flex h-6 items-center gap-1.5 rounded-full border px-2 text-xs font-medium whitespace-nowrap",
        className,
      )}
      style={{
        backgroundColor: `var(--status-${meta.token}-bg)`,
        color: `var(--status-${meta.token}-fg)`,
        borderColor: `var(--status-${meta.token}-border)`,
      }}
    >
      <Icon aria-hidden className="size-3.5" strokeWidth={2.25} />
      {meta.label}
    </span>
  );
}
