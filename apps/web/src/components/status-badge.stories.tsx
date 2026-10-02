import type { Meta, StoryObj } from "@storybook/react-vite";
import { PRIORITIES, WORK_ORDER_STATUSES } from "@/features/work-orders/status";
import { PriorityIndicator } from "./priority-indicator";
import { StatusBadge } from "./status-badge";

const meta = {
  title: "Domain/StatusBadge",
  component: StatusBadge,
  args: { status: "in_progress" },
  argTypes: { status: { control: "select", options: WORK_ORDER_STATUSES } },
} satisfies Meta<typeof StatusBadge>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Single: Story = {};

/** Every lifecycle state: color + icon + label, checked for contrast in both themes. */
export const AllStatuses: Story = {
  render: () => (
    <div className="flex flex-wrap gap-2">
      {WORK_ORDER_STATUSES.map((s) => (
        <StatusBadge key={s} status={s} />
      ))}
    </div>
  ),
};

export const Priorities: Story = {
  render: () => (
    <div className="flex flex-wrap gap-6 rounded-lg bg-surface-1 p-4">
      {PRIORITIES.map((p) => (
        <PriorityIndicator key={p} priority={p} />
      ))}
    </div>
  ),
};

/** Same set in dark theme, so a11y story tests verify both themes. */
export const AllStatusesDark: Story = {
  ...AllStatuses,
  globals: { theme: "dark" },
};

export const PrioritiesDark: Story = {
  ...Priorities,
  globals: { theme: "dark" },
};
