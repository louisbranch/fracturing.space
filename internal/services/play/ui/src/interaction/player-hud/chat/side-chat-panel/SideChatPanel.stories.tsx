import type { Meta, StoryObj } from "@storybook/react-vite";
import { SideChatPanelPreview } from "./SideChatPanelPreview";
import { emptySideChatState, sideChatState } from "./fixtures";

const meta = {
  title: "Interaction/Player HUD/Chat/Side Chat Panel",
  component: SideChatPanelPreview,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component: "Complete side chat panel with message list, compose input, and a right-side participant portrait rail for typing awareness.",
      },
    },
  },
  decorators: [
    (Story) => (
      <div className="flex h-dvh w-[30rem] flex-col">
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof SideChatPanelPreview>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Conversation: Story = {
  args: { initialState: sideChatState },
  parameters: {
    docs: {
      description: {
        story: "Standard Side Chat conversation with the participant rail visible and one participant currently typing.",
      },
    },
  },
};

export const EmptyChat: Story = {
  args: { initialState: emptySideChatState },
  parameters: {
    docs: {
      description: {
        story: "Empty Side Chat transcript with the portrait rail still present, useful for evaluating the idle participant list and empty-thread layout together.",
      },
    },
  },
};
