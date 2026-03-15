import type { Meta, StoryObj } from "@storybook/react-vite";
import { PhaseStatusBanner } from "./PhaseStatusBanner";
import { phaseStatusBannerFixtures } from "./fixtures";

const meta = {
  title: "Interaction/Legacy/Phase Status Banner",
  component: PhaseStatusBanner,
  tags: ["autodocs"],
  parameters: {
    docs: {
      description: {
        component:
          "Workflow-owned phase status summary for GM, players, GM review, and OOC-open states.",
      },
    },
  },
} satisfies Meta<typeof PhaseStatusBanner>;

export default meta;

type Story = StoryObj<typeof meta>;

export const PlayersOpen: Story = {
  args: {
    phase: phaseStatusBannerFixtures.players,
    viewerName: "Guide",
    viewerRole: "gm",
  },
};

export const GMReview: Story = {
  args: {
    phase: phaseStatusBannerFixtures.gmReview,
    viewerName: "Guide",
    viewerRole: "gm",
  },
};

export const OOCOpen: Story = {
  args: {
    phase: phaseStatusBannerFixtures.ooc,
    viewerName: "Guide",
    viewerRole: "gm",
  },
};
