import type { Meta, StoryObj } from "@storybook/react-vite";
import { BackstageContextCard } from "./BackstageContextCard";
import { backstageFixtureCatalog } from "./fixtures";

const meta = {
  title: "Interaction/Player HUD/Backstage/Context Card",
  component: BackstageContextCard,
  parameters: {
    docs: {
      description: {
        component: "Paused-play context block for Backstage, showing the active scene, paused prompt excerpt, and OOC reason without overloading the status banner.",
      },
    },
  },
} satisfies Meta<typeof BackstageContextCard>;

export default meta;

type Story = StoryObj<typeof meta>;

export const OpenDiscussion: Story = {
  args: {
    sceneName: backstageFixtureCatalog.openDiscussion.sceneName,
    pausedPromptText: backstageFixtureCatalog.openDiscussion.pausedPromptText,
    reason: backstageFixtureCatalog.openDiscussion.reason,
  },
  parameters: {
    docs: {
      description: {
        story: "Typical OOC pause context with the scene name, the paused player prompt, and the reason for the out-of-character stop.",
      },
    },
  },
};

export const WithoutReason: Story = {
  args: {
    sceneName: backstageFixtureCatalog.openDiscussion.sceneName,
    pausedPromptText: backstageFixtureCatalog.openDiscussion.pausedPromptText,
  },
  parameters: {
    docs: {
      description: {
        story: "Fallback context when the OOC reason is absent but the paused scene and prompt still need to anchor the discussion.",
      },
    },
  },
};
