import type { Meta, StoryObj } from "@storybook/react-vite";
import { onStageStatusBadge } from "../../shared/view-models";
import { archerGuardIllustration, onStageFixtureCatalog } from "./fixtures";
import { OnStageGMInteractionCard } from "./OnStageGMInteractionCard";

const meta = {
  title: "Interaction/Player HUD/On Stage/GM Interaction Card",
  component: OnStageGMInteractionCard,
  tags: ["autodocs"],
  parameters: {
    docs: {
      description: {
        component:
          "Structured GM interaction card for the stable On Stage surface, including beat history navigation and optional floated illustrations.",
      },
    },
  },
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

export const WithWideIllustration: Story = {
  args: {
    currentInteraction: onStageFixtureCatalog.viewerPosted.currentInteraction,
    interactionHistory: onStageFixtureCatalog.viewerPosted.interactionHistory,
    currentStatus: onStageStatusBadge(onStageFixtureCatalog.viewerPosted),
  },
};

export const WithCompactIllustration: Story = {
  args: {
    currentInteraction: onStageFixtureCatalog.viewerPosted.currentInteraction
      ? {
          ...onStageFixtureCatalog.viewerPosted.currentInteraction,
          illustration: archerGuardIllustration,
        }
      : undefined,
    interactionHistory: onStageFixtureCatalog.viewerPosted.interactionHistory,
    currentStatus: onStageStatusBadge(onStageFixtureCatalog.viewerPosted),
  },
};

export const NoIllustration: Story = {
  args: {
    currentInteraction: onStageFixtureCatalog.viewerPosted.currentInteraction
      ? {
          ...onStageFixtureCatalog.viewerPosted.currentInteraction,
          illustration: undefined,
        }
      : undefined,
    interactionHistory: onStageFixtureCatalog.viewerPosted.interactionHistory,
    currentStatus: onStageStatusBadge(onStageFixtureCatalog.viewerPosted),
  },
};
