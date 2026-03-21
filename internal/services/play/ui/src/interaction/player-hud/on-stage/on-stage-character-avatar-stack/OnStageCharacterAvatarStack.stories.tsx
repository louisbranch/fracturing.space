import type { Meta, StoryObj } from "@storybook/react-vite";
import { OnStageCharacterAvatarStack } from "./OnStageCharacterAvatarStack";
import { onStageFixtureCatalog } from "./fixtures";

const meta = {
  title: "Interaction/Player HUD/On Stage/Character Avatar Stack",
  component: OnStageCharacterAvatarStack,
  parameters: {
    docs: {
      description: {
        component:
          "Stacked character-avatar treatment for participant-owned On Stage slots, showing up to three portraits or two plus an ellipsis when more are involved.",
      },
    },
  },
  decorators: [
    (Story) => (
      <div className="flex min-h-32 items-center bg-base-100 px-6">
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof OnStageCharacterAvatarStack>;

export default meta;

type Story = StoryObj<typeof meta>;

export const SingleCharacter: Story = {
  args: {
    characters: onStageFixtureCatalog.viewerPosted.slots[0]?.characters ?? [],
  },
};

export const Overflow: Story = {
  args: {
    characters: onStageFixtureCatalog.multiCharacterOwner.slots[0]?.characters ?? [],
  },
};
