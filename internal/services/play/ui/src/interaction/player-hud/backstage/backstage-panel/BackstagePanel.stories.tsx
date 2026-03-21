import type { Meta, StoryObj } from "@storybook/react-vite";
import { BackstagePanelPreview } from "./BackstagePanelPreview";
import { backstageFixtureCatalog } from "./fixtures";

const meta = {
  title: "Interaction/Player HUD/Backstage/Panel",
  component: BackstagePanelPreview,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component: "Full player-facing Backstage surface for the authoritative OOC flow.",
      },
    },
  },
} satisfies Meta<typeof BackstagePanelPreview>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Dormant: Story = {
  args: { initialState: backstageFixtureCatalog.dormant },
  parameters: {
    docs: {
      description: {
        story: "Player-facing Backstage tab while OOC is closed. The surface stays visible, but posting and ready actions are inactive.",
      },
    },
  },
};

export const OpenEmpty: Story = {
  args: { initialState: backstageFixtureCatalog.openEmpty },
  parameters: {
    docs: {
      description: {
        story:
          "A newly opened OOC pause with the paused-scene context card and no transcript yet, useful for first-message composition and empty-state review.",
      },
    },
  },
};

export const OpenDiscussion: Story = {
  args: { initialState: backstageFixtureCatalog.openDiscussion },
  parameters: {
    docs: {
      description: {
        story:
          "The main active OOC state: terse status banner, paused-scene context card, discussion transcript, ready action, and the portrait rail with mixed participant statuses.",
      },
    },
  },
};

export const ViewerReady: Story = {
  args: { initialState: backstageFixtureCatalog.viewerReady },
  parameters: {
    docs: {
      description: {
        story: "The viewer has marked ready while the rest of the table is still resolving discussion, showing the intermediate handoff state.",
      },
    },
  },
};

export const WaitingOnGM: Story = {
  args: { initialState: backstageFixtureCatalog.waitingOnGM },
  parameters: {
    docs: {
      description: {
        story: "All players are ready and the panel is now waiting for the GM to resume on-stage play.",
      },
    },
  },
};
