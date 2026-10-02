import type { Meta, StoryObj } from "@storybook/react-vite";
import { ConnectionStatus } from "./connection-status";

const meta = {
  title: "Shell/ConnectionStatus",
  component: ConnectionStatus,
  args: { state: "connected" },
  argTypes: {
    state: { control: "inline-radio", options: ["connected", "reconnecting", "offline"] },
  },
} satisfies Meta<typeof ConnectionStatus>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Connected: Story = {};
export const Reconnecting: Story = { args: { state: "reconnecting" } };
export const Offline: Story = { args: { state: "offline" } };
