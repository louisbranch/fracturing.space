import type { Meta, StoryObj } from "@storybook/react-vite";
import { CharacterCard } from "./CharacterCard";
import { CharacterCardStoryStage } from "./StoryStage";
import { characterCardFixtures } from "./fixtures";

const meta = {
  title: "Systems/Daggerheart/Character Card",
  component: CharacterCard,
  tags: ["autodocs"],
  args: {
    character: characterCardFixtures.full,
  },
  parameters: {
    docs: {
      description: {
        component:
          "Compact character card with two display densities: portrait-only for spotlights, and a basic info card following the web campaign-card hierarchy.",
      },
    },
  },
} satisfies Meta<typeof CharacterCard>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Portrait: Story = {
  args: {
    variant: "portrait",
  },
  render: (args) => (
    <CharacterCardStoryStage variant="portrait">
      <CharacterCard {...args} />
    </CharacterCardStoryStage>
  ),
};

export const FullCard: Story = {
  args: {
    variant: "basic",
  },
  render: (args) => (
    <CharacterCardStoryStage variant="basic">
      <CharacterCard {...args} />
    </CharacterCardStoryStage>
  ),
};
