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
  tags: ["autodocs"],
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

const backstageAllStates = [
  { name: "Backstage Idle", state: backstageFixtureCatalog.dormant },
  { name: "OOC Open", state: backstageFixtureCatalog.openDiscussion },
  { name: "Ready", state: backstageFixtureCatalog.viewerReady },
  { name: "Waiting on GM", state: backstageFixtureCatalog.waitingOnGM },
] as const;

export const AllStates: Story = {
  args: {
    sceneName: backstageFixtureCatalog.openDiscussion.sceneName,
    pausedPromptText: backstageFixtureCatalog.openDiscussion.pausedPromptText,
    reason: backstageFixtureCatalog.openDiscussion.reason,
    status: statusArgs(backstageFixtureCatalog.openDiscussion),
  },
  render: () => (
    <div className="flex max-w-4xl flex-col gap-4">
      {backstageAllStates.map(({ name, state }) => (
        <div key={name} className="flex flex-col gap-2">
          <div className="preview-kicker">{name}</div>
          <BackstageContextCard
            sceneName={state.sceneName}
            pausedPromptText={state.pausedPromptText}
            reason={state.reason}
            status={statusArgs(state)}
          />
        </div>
      ))}
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: "Overview of every backstage status badge state, including the waiting-on-gm loading-bar treatment.",
      },
    },
  },
};
