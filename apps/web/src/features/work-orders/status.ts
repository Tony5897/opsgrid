import {
  CalendarCheckIcon,
  CircleCheckIcon,
  CircleDashedIcon,
  CirclePauseIcon,
  CircleSlashIcon,
  type LucideIcon,
  PlayIcon,
  SendIcon,
} from "lucide-react";

/**
 * Work-order lifecycle vocabulary (IMPLEMENTATION_PLAN §4.5). The server owns
 * transitions; the UI only renders states and offers commands the server
 * advertises. These literals will be replaced by generated OpenAPI types.
 */
export const WORK_ORDER_STATUSES = [
  "draft",
  "scheduled",
  "dispatched",
  "in_progress",
  "blocked",
  "completed",
  "cancelled",
] as const;
export type WorkOrderStatus = (typeof WORK_ORDER_STATUSES)[number];

export const PRIORITIES = ["low", "normal", "high", "urgent"] as const;
export type Priority = (typeof PRIORITIES)[number];

interface StatusMeta {
  label: string;
  icon: LucideIcon;
  /** CSS custom-property stem: --status-<token>-{bg,fg,border} */
  token: string;
  description: string;
}

export const STATUS_META: Record<WorkOrderStatus, StatusMeta> = {
  draft: {
    label: "Draft",
    icon: CircleDashedIcon,
    token: "draft",
    description: "Not yet scheduled",
  },
  scheduled: {
    label: "Scheduled",
    icon: CalendarCheckIcon,
    token: "scheduled",
    description: "Assigned to a technician and time window",
  },
  dispatched: {
    label: "Dispatched",
    icon: SendIcon,
    token: "dispatched",
    description: "Technician notified and en route",
  },
  in_progress: {
    label: "In progress",
    icon: PlayIcon,
    token: "in-progress",
    description: "Work has started on site",
  },
  blocked: {
    label: "Blocked",
    icon: CirclePauseIcon,
    token: "blocked",
    description: "Waiting on parts, access or approval",
  },
  completed: {
    label: "Completed",
    icon: CircleCheckIcon,
    token: "completed",
    description: "Work finished",
  },
  cancelled: {
    label: "Cancelled",
    icon: CircleSlashIcon,
    token: "cancelled",
    description: "Will not be performed",
  },
};

export const PRIORITY_META: Record<Priority, { label: string; bars: 1 | 2 | 3 | 4 }> = {
  low: { label: "Low", bars: 1 },
  normal: { label: "Normal", bars: 2 },
  high: { label: "High", bars: 3 },
  urgent: { label: "Urgent", bars: 4 },
};
