import type { Meta, StoryObj } from "@storybook/react-vite";
import { CharacterSheet } from "./CharacterSheet";
import { CharacterSheetStoryStage } from "./StoryStage";
import { characterSheetFixtures } from "./fixtures";

const meta = {
  title: "Systems/Daggerheart/Character Sheet",
  component: CharacterSheet,
  tags: ["autodocs"],
  parameters: {
    docs: {
      description: {
        component:
          "Read-only character sheet following the official Daggerheart paper sheet layout, using DaisyUI components for a better digital presentation. Display-only — no form inputs or editable state.",
      },
    },
  },
} satisfies Meta<typeof CharacterSheet>;

export default meta;

type Story = StoryObj<typeof meta>;

export const FullCharacter: Story = {
  args: {
    character: characterSheetFixtures.full,
  },
  render: (args) => (
    <CharacterSheetStoryStage>
      <CharacterSheet {...args} />
    </CharacterSheetStoryStage>
  ),
};

export const DamagedCharacter: Story = {
  args: {
    character: characterSheetFixtures.damaged,
  },
  render: (args) => (
    <CharacterSheetStoryStage>
      <CharacterSheet {...args} />
    </CharacterSheetStoryStage>
  ),
};
