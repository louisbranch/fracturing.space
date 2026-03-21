import type { Meta, StoryObj } from "@storybook/react-vite";
import { OnStageSlotCard } from "./OnStageSlotCard";
import { onStageFixtureCatalog } from "./fixtures";

const viewerState = onStageFixtureCatalog.viewerPosted;
const viewerSlot = viewerState.slots[0]!;
const viewerParticipant = viewerState.participants.find(
  (participant) => participant.id === viewerSlot?.participantId,
)!;
const revisionState = onStageFixtureCatalog.changesRequested;
const revisionSlot = revisionState.slots[0]!;
const revisionParticipant = revisionState.participants.find(
  (participant) => participant.id === revisionSlot?.participantId,
)!;

const meta = {
  title: "Interaction/Player HUD/On Stage/Slot Card",
  component: OnStageSlotCard,
  parameters: {
    docs: {
      description: {
        component:
          "Participant-owned On Stage slot card with stacked character avatars, visible character names, review state, and optional revision guidance.",
      },
    },
  },
} satisfies Meta<typeof OnStageSlotCard>;

export default meta;

type Story = StoryObj<typeof meta>;

export const ViewerSlot: Story = {
  args: {
    slot: viewerSlot,
    participant: viewerParticipant,
    isViewer: true,
  },
};

export const ChangesRequested: Story = {
  args: {
    slot: revisionSlot,
    participant: revisionParticipant,
    isViewer: true,
  },
};
