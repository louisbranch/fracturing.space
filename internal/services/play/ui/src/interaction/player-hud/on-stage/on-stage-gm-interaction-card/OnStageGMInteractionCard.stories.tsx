import type { Meta, StoryObj } from "@storybook/react-vite";
import { onStageStatusBadge } from "../../shared/view-models";
import { onStageFixtureCatalog } from "./fixtures";
import { OnStageGMInteractionCard } from "./OnStageGMInteractionCard";

const meta = {
  title: "Interaction/Player HUD/On Stage/GM Interaction Card",
  component: OnStageGMInteractionCard,
  tags: ["autodocs"],
} satisfies Meta<typeof OnStageGMInteractionCard>;

export default meta;

type Story = StoryObj<typeof meta>;

export const CurrentInteraction: Story = {
  args: {
    currentInteraction: onStageFixtureCatalog.viewerPosted.currentInteraction,
    interactionHistory: onStageFixtureCatalog.viewerPosted.interactionHistory,
    currentStatus: onStageStatusBadge(onStageFixtureCatalog.viewerPosted),
  },
};

export const WaitingOnGM: Story = {
  args: {
    currentInteraction: onStageFixtureCatalog.aiThinking.currentInteraction,
    interactionHistory: onStageFixtureCatalog.aiThinking.interactionHistory,
    currentStatus: onStageStatusBadge(onStageFixtureCatalog.aiThinking),
  },
};
