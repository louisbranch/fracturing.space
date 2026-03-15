import type { Meta, StoryObj } from "@storybook/react-vite";
import { GMReviewPanel } from "./GMReviewPanel";
import { gmReviewPanelFixtures } from "./fixtures";

const meta = {
  title: "Interaction/Legacy/GM Review Panel",
  component: GMReviewPanel,
  tags: ["autodocs"],
  parameters: {
    docs: {
      description: {
        component:
          "GM-only beat review summary and action cluster for accept, revise, or end-phase decisions.",
      },
    },
  },
} satisfies Meta<typeof GMReviewPanel>;

export default meta;

type Story = StoryObj<typeof meta>;

export const ReadyToResolve: Story = {
  args: {
    slots: gmReviewPanelFixtures.review,
  },
};

export const RevisionsAlreadyRequested: Story = {
  args: {
    slots: gmReviewPanelFixtures.revisions,
  },
};
