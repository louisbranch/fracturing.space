import type { Meta, StoryObj } from "@storybook/react-vite";
import { ChatSidecar } from "./ChatSidecar";
import { chatSidecarFixtures } from "./fixtures";

const meta = {
  title: "Interaction/Legacy/Chat Sidecar",
  component: ChatSidecar,
  tags: ["autodocs"],
  parameters: {
    docs: {
      description: {
        component:
          "Optional human transcript sidecar kept intentionally separate from authoritative interaction state.",
      },
    },
  },
} satisfies Meta<typeof ChatSidecar>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    messages: chatSidecarFixtures.messages,
  },
};
