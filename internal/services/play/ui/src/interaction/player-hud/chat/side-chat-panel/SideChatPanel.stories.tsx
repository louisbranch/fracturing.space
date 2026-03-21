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
        component: "Complete side chat panel with message list and compose input.",
      },
    },
  },
  decorators: [
    (Story) => (
      <div className="flex h-dvh w-80 flex-col">
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof SideChatPanelPreview>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Conversation: Story = {
  args: { initialState: sideChatState },
};

export const EmptyChat: Story = {
  args: { initialState: emptySideChatState },
};
