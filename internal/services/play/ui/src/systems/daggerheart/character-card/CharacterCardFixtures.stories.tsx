import type { Meta, StoryObj } from "@storybook/react-vite";
import { CharacterCard } from "./CharacterCard";
import { CharacterCardStoryStage } from "./StoryStage";
import { characterCardFixtures } from "./fixtures";

const meta = {
  title: "Systems/Daggerheart/Character Card/Fixtures",
  component: CharacterCard,
  tags: ["autodocs"],
  parameters: {
    docs: {
      description: {
        component:
          "Fixture stories keep the Character Card contract constant and vary only the canonical web-derived mock data. The canvas shows the component directly without in-canvas explanation panels.",
      },
    },
  },
} satisfies Meta<typeof CharacterCard>;

export default meta;

type Story = StoryObj<typeof meta>;

export const FullFixture: Story = {
  args: {
    character: characterCardFixtures.full,
    variant: "full",
  },
  render: (args) => (
    <CharacterCardStoryStage variant="full">
      <CharacterCard {...args} />
    </CharacterCardStoryStage>
  ),
};

export const MinimalFixture: Story = {
  args: {
    character: characterCardFixtures.minimal,
    variant: "basic",
  },
  render: (args) => (
    <CharacterCardStoryStage variant="basic">
      <CharacterCard {...args} />
    </CharacterCardStoryStage>
  ),
};

export const PartialFixture: Story = {
  args: {
    character: characterCardFixtures.partial,
    variant: "full",
  },
  render: (args) => (
    <CharacterCardStoryStage variant="full">
      <CharacterCard {...args} />
    </CharacterCardStoryStage>
  ),
};
