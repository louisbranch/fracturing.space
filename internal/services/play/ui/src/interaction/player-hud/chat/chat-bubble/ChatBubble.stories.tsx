import type { Meta, StoryObj } from "@storybook/react-vite";
import { ChatBubble } from "./ChatBubble";

const meta = {
  title: "Interaction/Player HUD/Chat/Chat Bubble",
  component: ChatBubble,
  tags: ["autodocs"],
  parameters: {
    docs: {
      description: {
        component: "A single message bubble in the side chat panel.",
      },
    },
  },
} satisfies Meta<typeof ChatBubble>;

export default meta;

type Story = StoryObj<typeof meta>;

export const OwnMessage: Story = {
  args: {
    body: "Copy. Moving to the bridge.",
    time: "16:31",
    alignment: "end",
  },
};

export const OtherMessage: Story = {
  args: {
    body: "Ready when you are.",
    time: "16:30",
    alignment: "start",
    showName: "Corin",
    showAvatar: true,
    avatarFallback: "C",
  },
};

export const FirstInRun: Story = {
  args: {
    body: "Ready when you are.",
    time: "16:30",
    alignment: "start",
    showName: "Corin",
  },
};

export const LastInRun: Story = {
  args: {
    body: "I'll take the left flank.",
    time: "16:30",
    alignment: "start",
    showAvatar: true,
    avatarFallback: "C",
  },
};
