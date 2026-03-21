import type { Meta, StoryObj } from "@storybook/react-vite";
import { BackstageStatusBanner } from "./BackstageStatusBanner";
import { backstageFixtureCatalog } from "./fixtures";

const meta = {
  title: "Interaction/Player HUD/Backstage/Status Banner",
  component: BackstageStatusBanner,
  args: {
    onViewerReadyToggle: () => {},
  },
  parameters: {
    docs: {
      description: {
        component:
          "Top status banner for the Backstage OOC flow, reduced to terse state copy and the player ready action. Scene and pause context live in the separate context card.",
      },
    },
  },
} satisfies Meta<typeof BackstageStatusBanner>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Dormant: Story = {
  args: {
    mode: backstageFixtureCatalog.dormant.mode,
    resumeState: backstageFixtureCatalog.dormant.resumeState,
    viewerReady: false,
  },
  parameters: {
    docs: {
      description: {
        story: "Backstage is visible but OOC is not open yet, so the banner acts as an inactive explainer and the ready control stays disabled.",
      },
    },
  },
};

export const OpenDiscussion: Story = {
  args: {
    mode: backstageFixtureCatalog.openDiscussion.mode,
    resumeState: backstageFixtureCatalog.openDiscussion.resumeState,
    viewerReady: false,
  },
  parameters: {
    docs: {
      description: {
        story:
          "OOC is active and still collecting readiness, so the banner stays terse while the context card below carries the paused-scene details.",
      },
    },
  },
};

export const WaitingOnGM: Story = {
  args: {
    mode: backstageFixtureCatalog.waitingOnGM.mode,
    resumeState: backstageFixtureCatalog.waitingOnGM.resumeState,
    viewerReady: true,
  },
  parameters: {
    docs: {
      description: {
        story: "All players are already ready, so the banner shifts to a waiting-for-GM state while leaving the player’s ready state visible.",
      },
    },
  },
};
