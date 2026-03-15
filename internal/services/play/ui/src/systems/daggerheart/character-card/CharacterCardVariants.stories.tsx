import type { Meta, StoryObj } from "@storybook/react-vite";
import { CharacterCard } from "./CharacterCard";
import { CharacterCardStoryStage } from "./StoryStage";
import { characterCardFixtures } from "./fixtures";

const meta = {
  title: "Systems/Daggerheart/Character Card/Variants",
  component: CharacterCard,
  tags: ["autodocs"],
  args: {
    character: characterCardFixtures.full,
  },
  parameters: {
    docs: {
      description: {
        component:
          "Variant stories hold the fixture steady and change only the documented display density. The canvas now shows only the component in a lightweight screen frame.",
      },
    },
  },
} satisfies Meta<typeof CharacterCard>;

export default meta;

type Story = StoryObj<typeof meta>;

export const PortraitOnly: Story = {
  args: {
    variant: "portrait",
  },
  render: (args) => (
    <CharacterCardStoryStage variant="portrait">
      <CharacterCard {...args} />
    </CharacterCardStoryStage>
  ),
};

export const BasicInfo: Story = {
  args: {
    variant: "basic",
  },
  render: (args) => (
    <CharacterCardStoryStage variant="basic">
      <CharacterCard {...args} />
    </CharacterCardStoryStage>
  ),
};

export const FullInfo: Story = {
  args: {
    variant: "full",
  },
  render: (args) => (
    <CharacterCardStoryStage variant="full">
      <CharacterCard {...args} />
    </CharacterCardStoryStage>
  ),
};
