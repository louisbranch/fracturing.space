import type { Meta, StoryObj } from "@storybook/react-vite";
import { HUDConnectionBadge } from "./HUDConnectionBadge";
import { hudConnectionBadgeFixtures } from "./fixtures";

const meta = {
  title: "Interaction/Player HUD/Shared/Connection Badge",
  component: HUDConnectionBadge,
  tags: ["autodocs"],
  parameters: {
    docs: {
      description: {
        component: "Passive transport-health badge for the Player HUD navbar, with connected, reconnecting, and disconnected states.",
      },
    },
  },
} satisfies Meta<typeof HUDConnectionBadge>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Connected: Story = {
  args: hudConnectionBadgeFixtures.connected,
};

export const Reconnecting: Story = {
  args: hudConnectionBadgeFixtures.reconnecting,
};

export const Disconnected: Story = {
  args: hudConnectionBadgeFixtures.disconnected,
};
