import type { Meta, StoryObj } from "@storybook/react-vite";
import { OnStagePanelPreview } from "./OnStagePanelPreview";
import { onStagePanelFixtures } from "./fixtures";

const meta = {
  title: "Interaction/Player HUD/On Stage/Panel",
  component: OnStagePanelPreview,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Full player-facing On Stage surface for the authoritative scene interaction flow, including acting slots, revision states, and blocked wait states.",
      },
    },
  },
} satisfies Meta<typeof OnStagePanelPreview>;

export default meta;

type Story = StoryObj<typeof meta>;

export const WaitingOnGM: Story = {
  args: { initialState: onStagePanelFixtures.waitingOnGM },
};

export const ActingEmpty: Story = {
  args: { initialState: onStagePanelFixtures.actingEmpty },
};

export const ViewerPosted: Story = {
  args: { initialState: onStagePanelFixtures.viewerPosted },
};

export const YieldedWaiting: Story = {
  args: { initialState: onStagePanelFixtures.yieldedWaiting },
};

export const ChangesRequested: Story = {
  args: { initialState: onStagePanelFixtures.changesRequested },
};

export const OOCBlocked: Story = {
  args: { initialState: onStagePanelFixtures.oocBlocked },
};

export const AIThinking: Story = {
  args: { initialState: onStagePanelFixtures.aiThinking },
};

export const AIFailed: Story = {
  args: { initialState: onStagePanelFixtures.aiFailed },
};

export const MultiCharacterOwner: Story = {
  args: { initialState: onStagePanelFixtures.multiCharacterOwner },
};
