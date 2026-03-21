import type { Meta, StoryObj } from "@storybook/react-vite";
import { ParticipantPortraitRail } from "./ParticipantPortraitRail";
import { participantPortraitRailFixtures } from "./fixtures";

const meta = {
  title: "Interaction/Player HUD/Shared/Participant Portrait Rail",
  component: ParticipantPortraitRail,
  parameters: {
    docs: {
      description: {
        component:
          "Shared right-side portrait rail for Player HUD surfaces, with tooltip-backed idle, typing, ready, and On Stage progress overlays plus an optional GM-authority marker.",
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
} satisfies Meta<typeof ParticipantPortraitRail>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Typing: Story = {
  args: {
    participants: participantPortraitRailFixtures.typing,
    viewerParticipantId: "p-rhea",
    ariaLabel: "Side chat participants",
  },
  parameters: {
    docs: {
      description: {
        story:
          "Typing-focused rail state for Side Chat. The badges explain participant status on hover, while the GM portrait keeps the authority marker.",
      },
    },
  },
};

export const Ready: Story = {
  args: {
    participants: participantPortraitRailFixtures.ready,
    viewerParticipantId: "p-rhea",
    ariaLabel: "Backstage participants",
  },
  parameters: {
    docs: {
      description: {
        story:
          "Backstage-style readiness state where player portraits can show ready badges and the GM authority owner is called out separately.",
      },
    },
  },
};

export const Active: Story = {
  args: {
    participants: participantPortraitRailFixtures.active,
    viewerParticipantId: "p-rhea",
    ariaLabel: "On-stage participants",
  },
};

export const ChangesRequested: Story = {
  args: {
    participants: participantPortraitRailFixtures.changesRequested,
    viewerParticipantId: "p-rhea",
    ariaLabel: "On-stage participants",
  },
};
