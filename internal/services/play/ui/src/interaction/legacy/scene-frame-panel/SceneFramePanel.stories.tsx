import type { Meta, StoryObj } from "@storybook/react-vite";
import { interactionComponentFixtures } from "../shared/fixtures";
import { SceneFramePanel } from "./SceneFramePanel";
import { sceneFramePanelFixtures } from "./fixtures";

const meta = {
  title: "Interaction/Legacy/Scene Frame Panel",
  component: SceneFramePanel,
  tags: ["autodocs"],
  parameters: {
    docs: {
      description: {
        component:
          "Active scene narrative panel combining scene identity, phase context, committed GM output, and the current player-facing frame.",
      },
    },
  },
} satisfies Meta<typeof SceneFramePanel>;

export default meta;

type Story = StoryObj<typeof meta>;

export const ActiveScene: Story = {
  args: {
    phase: interactionComponentFixtures.phase.players,
    scene: sceneFramePanelFixtures.activeScene,
  },
};

export const NoSceneSelected: Story = {
  args: {
    phase: interactionComponentFixtures.phase.players,
    scene: sceneFramePanelFixtures.emptyScene,
  },
};
