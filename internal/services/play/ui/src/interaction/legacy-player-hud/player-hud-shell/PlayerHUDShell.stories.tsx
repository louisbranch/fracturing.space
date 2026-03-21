import type { Meta, StoryObj } from "@storybook/react-vite";
import { PlayerHUDShellPreview } from "./PlayerHUDShellPreview";
import { playerHUDShellFixtures } from "./fixtures";

const meta = {
  title: "Interaction/Legacy Player HUD/Shell",
  component: PlayerHUDShellPreview,
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof PlayerHUDShellPreview>;

export default meta;

type Story = StoryObj<typeof meta>;

export const PlayerTurn: Story = {
  args: {
    initialState: playerHUDShellFixtures.playerTurn,
  },
};

export const OOCPaused: Story = {
  args: {
    initialState: playerHUDShellFixtures.oocPaused,
  },
};

export const Reconnecting: Story = {
  args: {
    initialState: playerHUDShellFixtures.reconnecting,
  },
};

export const CollapsedComposer: Story = {
  args: {
    initialState: playerHUDShellFixtures.collapsed,
  },
};
