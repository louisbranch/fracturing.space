import type { Meta, StoryObj } from "@storybook/react-vite";
import { StageViewport } from "./StageViewport";
import { stageViewportFixtures } from "./fixtures";

const meta = {
  title: "Interaction/Player HUD/Stage Viewport",
  component: StageViewport,
  tags: ["autodocs"],
  parameters: {
    docs: {
      description: {
        component:
          "Central stage slot for future narration and prompt rendering, with internal scroll behavior instead of page scroll.",
      },
    },
  },
} satisfies Meta<typeof StageViewport>;

export default meta;

type Story = StoryObj<typeof meta>;

export const DefaultStage: Story = {
  args: {
    stage: stageViewportFixtures.default,
  },
};

export const ScrollingContent: Story = {
  args: {
    stage: stageViewportFixtures.scrolling,
  },
};

export const EmptyStage: Story = {
  args: {
    stage: stageViewportFixtures.empty,
  },
};
