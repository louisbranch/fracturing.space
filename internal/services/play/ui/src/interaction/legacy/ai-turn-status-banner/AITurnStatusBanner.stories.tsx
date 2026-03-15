import type { Meta, StoryObj } from "@storybook/react-vite";
import { AITurnStatusBanner } from "./AITurnStatusBanner";
import { aiTurnStatusBannerFixtures } from "./fixtures";

const meta = {
  title: "Interaction/Legacy/AI Turn Status Banner",
  component: AITurnStatusBanner,
  tags: ["autodocs"],
  parameters: {
    docs: {
      description: {
        component:
          "Compact cross-service AI GM lifecycle surface for queued, running, and failed turn states.",
      },
    },
  },
} satisfies Meta<typeof AITurnStatusBanner>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Queued: Story = {
  args: {
    aiTurn: aiTurnStatusBannerFixtures.queued,
  },
};

export const Failed: Story = {
  args: {
    aiTurn: aiTurnStatusBannerFixtures.failed,
  },
};
