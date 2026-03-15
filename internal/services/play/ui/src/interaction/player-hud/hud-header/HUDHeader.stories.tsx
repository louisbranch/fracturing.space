import type { Meta, StoryObj } from "@storybook/react-vite";
import { HUDHeader } from "./HUDHeader";
import { hudHeaderFixtures } from "./fixtures";

const meta = {
  title: "Interaction/Player HUD/Header",
  component: HUDHeader,
  tags: ["autodocs"],
  parameters: {
    docs: {
      description: {
        component:
          "Thin player HUD header with campaign context, a back link into the web campaign page, and realtime connection state.",
      },
    },
  },
} satisfies Meta<typeof HUDHeader>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Connected: Story = {
  args: {
    campaignName: hudHeaderFixtures.connected.campaignName,
    backURL: hudHeaderFixtures.connected.backURL,
    connection: hudHeaderFixtures.connected.connection,
  },
};

export const Reconnecting: Story = {
  args: {
    campaignName: hudHeaderFixtures.reconnecting.campaignName,
    backURL: hudHeaderFixtures.reconnecting.backURL,
    connection: hudHeaderFixtures.reconnecting.connection,
  },
};

export const Disconnected: Story = {
  args: {
    campaignName: hudHeaderFixtures.disconnected.campaignName,
    backURL: hudHeaderFixtures.disconnected.backURL,
    connection: hudHeaderFixtures.disconnected.connection,
  },
};
