import type { Meta, StoryObj } from "@storybook/react-vite";
import { SideChatParticipantRail } from "./SideChatParticipantRail";
import { emptySideChatState, sideChatState } from "./fixtures";

const meta = {
  title: "Interaction/Player HUD/Chat/Side Chat Participant Rail",
  component: SideChatParticipantRail,
  parameters: {
    docs: {
      description: {
        component:
          "Side Chat adapter for the shared participant portrait rail, preserving participant role labels while using typing overlays to show who is composing in the chat sidecar.",
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
} satisfies Meta<typeof SideChatParticipantRail>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Conversation: Story = {
  args: {
    participants: sideChatState.participants,
    viewerParticipantId: sideChatState.viewerParticipantId,
  },
  parameters: {
    docs: {
      description: {
        story:
          "Active Side Chat with one participant currently typing, shown as a typing overlay on the portrait rail while keeping the same PLAYER and GM labels as the other HUD tabs.",
      },
    },
  },
};

export const Idle: Story = {
  args: {
    participants: emptySideChatState.participants.map((participant) => ({ ...participant, typing: false })),
    viewerParticipantId: emptySideChatState.viewerParticipantId,
  },
  parameters: {
    docs: {
      description: {
        story: "No one is typing, so the side chat rail falls back to neutral portrait states.",
      },
    },
  },
};
