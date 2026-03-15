import type { Meta, StoryObj } from "@storybook/react-vite";
import { PlayInteractionShell } from "./PlayInteractionShell";
import { playInteractionShellFixtures } from "./fixtures";

const meta = {
  title: "Interaction/Legacy/Play Interaction Shell",
  component: PlayInteractionShell,
  tags: ["autodocs"],
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Composition-only Storybook shell that assembles the isolated interaction slices without implying any live runtime integration.",
      },
    },
  },
} satisfies Meta<typeof PlayInteractionShell>;

export default meta;

type Story = StoryObj<typeof meta>;

export const PlayersOpen: Story = {
  args: playInteractionShellFixtures.playersOpen,
};

export const OOCPause: Story = {
  args: playInteractionShellFixtures.oocOpen,
};

export const AIFailure: Story = {
  args: playInteractionShellFixtures.aiTurnFailed,
};
