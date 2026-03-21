import type { Meta, StoryObj } from "@storybook/react-vite";
import { PlayerHUDShellPreview } from "./PlayerHUDShellPreview";
import { playerHUDShellFixtures } from "./fixtures";

const meta = {
  title: "Interaction/Player HUD/Shell",
  component: PlayerHUDShellPreview,
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof PlayerHUDShellPreview>;

export default meta;

type Story = StoryObj<typeof meta>;

export const OnStage: Story = {
  args: { initialState: playerHUDShellFixtures.onStage },
};

export const Backstage: Story = {
  args: { initialState: playerHUDShellFixtures.backstage },
};

export const SideChat: Story = {
  args: { initialState: playerHUDShellFixtures.sideChat },
};
