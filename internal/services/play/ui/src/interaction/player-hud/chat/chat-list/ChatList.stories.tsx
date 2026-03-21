import type { Meta, StoryObj } from "@storybook/react-vite";
import { ChatList } from "./ChatList";
import { sideChatMessages, sideChatParticipants } from "./fixtures";

const meta = {
  title: "Interaction/Player HUD/Chat/Chat List",
  component: ChatList,
  tags: ["autodocs"],
  parameters: {
    docs: {
      description: {
        component: "Grouped message list content rendered inside the side chat panel's scroll region.",
      },
    },
  },
  decorators: [
    (Story) => (
      <div className="flex h-96 w-80 flex-col">
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof ChatList>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Conversation: Story = {
  args: {
    messages: sideChatMessages,
    participants: sideChatParticipants,
    viewerParticipantId: "p-rhea",
  },
};

export const EmptyChat: Story = {
  args: {
    messages: [],
    participants: sideChatParticipants,
    viewerParticipantId: "p-rhea",
  },
};
