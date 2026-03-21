import type { Meta, StoryObj } from "@storybook/react-vite";
import { ChatCompose } from "./ChatCompose";

const meta = {
  title: "Interaction/Player HUD/Chat/Chat Compose",
  component: ChatCompose,
  tags: ["autodocs"],
  args: {
    onDraftChange: () => {},
    onSend: () => {},
  },
  parameters: {
    docs: {
      description: {
        component: "Compose input with send button for the side chat panel.",
      },
    },
  },
} satisfies Meta<typeof ChatCompose>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Empty: Story = {
  args: { draft: "" },
};

export const WithDraft: Story = {
  args: { draft: "Should we prep anything?" },
};
