import type { Meta, StoryObj } from "@storybook/react-vite";
import { PlusIcon, TrashIcon } from "lucide-react";
import { expect, fn, userEvent, within } from "storybook/test";
import { Button } from "./button";

const meta = {
  title: "Primitives/Button",
  component: Button,
  args: { children: "Create work order", onClick: fn() },
  argTypes: {
    variant: { control: "select", options: ["primary", "secondary", "ghost", "danger", "link"] },
    size: { control: "select", options: ["sm", "md", "icon", "icon-sm"] },
  },
} satisfies Meta<typeof Button>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Primary: Story = {
  args: { variant: "primary" },
  play: async ({ canvasElement, args }) => {
    const button = within(canvasElement).getByRole("button", { name: "Create work order" });
    await expect(button).toHaveAttribute("type", "button");
    await userEvent.keyboard("{Tab}");
    await expect(button).toHaveFocus();
    await userEvent.keyboard("{Enter}");
    await expect(args.onClick).toHaveBeenCalledOnce();
  },
};

export const Variants: Story = {
  render: (args) => (
    <div className="flex flex-wrap items-center gap-3">
      <Button {...args} variant="primary">
        <PlusIcon aria-hidden /> Create
      </Button>
      <Button {...args} variant="secondary">
        Secondary
      </Button>
      <Button {...args} variant="ghost">
        Ghost
      </Button>
      <Button {...args} variant="danger">
        <TrashIcon aria-hidden /> Delete
      </Button>
      <Button {...args} variant="link">
        Link
      </Button>
      <Button {...args} variant="primary" disabled>
        Disabled
      </Button>
    </div>
  ),
};

export const IconOnly: Story = {
  args: {
    variant: "secondary",
    size: "icon",
    "aria-label": "Add",
    children: <PlusIcon aria-hidden />,
  },
};
