import type { Meta, StoryObj } from "@storybook/react-vite";
import { backstageStatusDisplay } from "../shared/status-display";
import { BackstageContextCard } from "./BackstageContextCard";
import { backstageFixtureCatalog } from "./fixtures";

function statusArgs(state: typeof backstageFixtureCatalog.openDiscussion) {
  const viewer = state.participants.find((participant) => participant.id === state.viewerParticipantId);
  return backstageStatusDisplay({
    mode: state.mode,
    resumeState: state.resumeState,
    viewerReady: Boolean(viewer?.readyToResume),
  });
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
    statusLabel: statusArgs(backstageFixtureCatalog.openDiscussion).badgeLabel,
    statusClassName: statusArgs(backstageFixtureCatalog.openDiscussion).badgeClassName,
    statusTooltip: statusArgs(backstageFixtureCatalog.openDiscussion).message,
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
    statusLabel: statusArgs(backstageFixtureCatalog.openDiscussion).badgeLabel,
    statusClassName: statusArgs(backstageFixtureCatalog.openDiscussion).badgeClassName,
    statusTooltip: statusArgs(backstageFixtureCatalog.openDiscussion).message,
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
    statusLabel: statusArgs(backstageFixtureCatalog.dormant).badgeLabel,
    statusClassName: statusArgs(backstageFixtureCatalog.dormant).badgeClassName,
    statusTooltip: statusArgs(backstageFixtureCatalog.dormant).message,
  },
};
