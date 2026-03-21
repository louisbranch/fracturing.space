import type { Meta, StoryObj } from "@storybook/react-vite";
import { backstageStatusBadge } from "../../shared/view-models";
import { BackstageContextCard } from "./BackstageContextCard";
import { backstageFixtureCatalog } from "./fixtures";

function statusArgs(state: typeof backstageFixtureCatalog.openDiscussion) {
  return backstageStatusBadge(state);
}

const meta = {
  title: "Interaction/Player HUD/Backstage/Context Card",
  component: BackstageContextCard,
  parameters: {
    docs: {
      description: {
        component: "Paused-play context block for Backstage, showing the active scene, paused prompt excerpt, OOC reason, and the current Backstage status badge in one place.",
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
    status: statusArgs(backstageFixtureCatalog.openDiscussion),
  },
  parameters: {
    docs: {
      description: {
        story: "Typical OOC pause context with the scene name, the paused player prompt, the reason for the stop, and the current status badge.",
      },
    },
  },
};

export const WithoutReason: Story = {
  args: {
    sceneName: backstageFixtureCatalog.openDiscussion.sceneName,
    pausedPromptText: backstageFixtureCatalog.openDiscussion.pausedPromptText,
    status: statusArgs(backstageFixtureCatalog.openDiscussion),
  },
  parameters: {
    docs: {
      description: {
        story: "Fallback context when the OOC reason is absent but the paused scene and prompt still need to anchor the discussion.",
      },
    },
  },
};

export const Dormant: Story = {
  args: {
    sceneName: backstageFixtureCatalog.dormant.sceneName,
    pausedPromptText: backstageFixtureCatalog.dormant.pausedPromptText,
    status: statusArgs(backstageFixtureCatalog.dormant),
  },
};
