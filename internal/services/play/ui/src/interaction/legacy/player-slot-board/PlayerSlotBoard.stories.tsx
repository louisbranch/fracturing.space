import type { Meta, StoryObj } from "@storybook/react-vite";
import { PlayerSlotBoard } from "./PlayerSlotBoard";
import { playerSlotBoardFixtures } from "./fixtures";

const meta = {
  title: "Interaction/Legacy/Player Slot Board",
  component: PlayerSlotBoard,
  tags: ["autodocs"],
  parameters: {
    docs: {
      description: {
        component:
          "Participant-owned slot summaries for open player beats and GM review states.",
      },
    },
  },
} satisfies Meta<typeof PlayerSlotBoard>;

export default meta;

type Story = StoryObj<typeof meta>;

export const OpenSlots: Story = {
  args: {
    slots: playerSlotBoardFixtures.open,
  },
};

export const UnderReview: Story = {
  args: {
    slots: playerSlotBoardFixtures.review,
  },
};

export const ChangesRequested: Story = {
  args: {
    slots: playerSlotBoardFixtures.revisions,
  },
};
