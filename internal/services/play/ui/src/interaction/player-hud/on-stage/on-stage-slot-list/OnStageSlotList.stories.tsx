import type { Meta, StoryObj } from "@storybook/react-vite";
import { OnStageSlotList } from "./OnStageSlotList";
import { onStageFixtureCatalog } from "./fixtures";

const meta = {
  title: "Interaction/Player HUD/On Stage/Slot List",
  component: OnStageSlotList,
  parameters: {
    docs: {
      description: {
        component:
          "Scrollable On Stage slot list that keeps one card per acting participant, even before that participant has committed a post.",
      },
    },
  },
  decorators: [
    (Story) => (
      <div className="flex h-dvh max-w-3xl flex-col bg-base-100">
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof OnStageSlotList>;

export default meta;

type Story = StoryObj<typeof meta>;

export const ActingEmpty: Story = {
  args: {
    participants: onStageFixtureCatalog.actingEmpty.participants,
    slots: onStageFixtureCatalog.actingEmpty.slots,
    actingParticipantIds: onStageFixtureCatalog.actingEmpty.actingParticipantIds,
    viewerParticipantId: onStageFixtureCatalog.actingEmpty.viewerParticipantId,
  },
};

export const MultiCharacterOwner: Story = {
  args: {
    participants: onStageFixtureCatalog.multiCharacterOwner.participants,
    slots: onStageFixtureCatalog.multiCharacterOwner.slots,
    actingParticipantIds: onStageFixtureCatalog.multiCharacterOwner.actingParticipantIds,
    viewerParticipantId: onStageFixtureCatalog.multiCharacterOwner.viewerParticipantId,
  },
};
