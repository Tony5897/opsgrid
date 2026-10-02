import type { Meta, StoryObj } from "@storybook/react-vite";
import { InboxIcon, PlusIcon } from "lucide-react";
import { EmptyState } from "./empty-state";
import { Button } from "./ui/button";

const meta = {
  title: "Patterns/EmptyState",
  component: EmptyState,
  args: {
    icon: InboxIcon,
    title: "No unassigned work",
    description: "Everything is scheduled. New work orders appear here as they come in.",
  },
} satisfies Meta<typeof EmptyState>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const WithAction: Story = {
  args: {
    action: (
      <Button variant="primary" size="sm">
        <PlusIcon aria-hidden /> Create work order
      </Button>
    ),
  },
};
