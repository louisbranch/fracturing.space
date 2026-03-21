import type { Meta, StoryObj } from "@storybook/react-vite";
import { OnStageParticipantRail } from "./OnStageParticipantRail";
import { onStageFixtureCatalog } from "./fixtures";

const meta = {
  title: "Interaction/Player HUD/On Stage/Participant Rail",
  component: OnStageParticipantRail,
  parameters: {
    docs: {
      description: {
        component:
          "Right-side portrait rail for On Stage, showing who is active, yielded, or revising while preserving GM authority visibility.",
      },
    },
  },
  decorators: [
    (Story) => (
      <div className="flex h-dvh justify-end bg-base-100">
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof OnStageParticipantRail>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Acting: Story = {
  args: {
    participants: onStageFixtureCatalog.viewerPosted.participants,
    viewerParticipantId: onStageFixtureCatalog.viewerPosted.viewerParticipantId,
  },
};

export const YieldedWaiting: Story = {
  args: {
    participants: onStageFixtureCatalog.yieldedWaiting.participants,
    viewerParticipantId: onStageFixtureCatalog.yieldedWaiting.viewerParticipantId,
  },
};

export const ChangesRequested: Story = {
  args: {
    participants: onStageFixtureCatalog.changesRequested.participants,
    viewerParticipantId: onStageFixtureCatalog.changesRequested.viewerParticipantId,
  },
};
