import type { Meta, StoryObj } from "@storybook/react-vite";
import { PlayerComposerPreview } from "./PlayerComposerPreview";
import { playerComposerFixtures } from "./fixtures";

const meta = {
  title: "Interaction/Player HUD/Composer",
  component: PlayerComposerPreview,
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof PlayerComposerPreview>;

export default meta;

type Story = StoryObj<typeof meta>;

export const ScratchPad: Story = {
  args: {
    initialState: playerComposerFixtures.gmTurn,
    showActionLog: true,
  },
};

export const PlayerTurn: Story = {
  args: {
    initialState: playerComposerFixtures.playerTurn,
    showActionLog: true,
  },
};

export const OOCPaused: Story = {
  args: {
    initialState: playerComposerFixtures.oocPaused,
    showActionLog: true,
  },
};

export const Minimized: Story = {
  args: {
    initialState: playerComposerFixtures.collapsed,
    showActionLog: true,
  },
};
