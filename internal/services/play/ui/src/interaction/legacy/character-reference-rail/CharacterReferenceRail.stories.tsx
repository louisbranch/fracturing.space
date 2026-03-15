import type { Meta, StoryObj } from "@storybook/react-vite";
import { CharacterReferenceRail } from "./CharacterReferenceRail";
import { characterReferenceRailFixtures } from "./fixtures";

const meta = {
  title: "Interaction/Legacy/Character Reference Rail",
  component: CharacterReferenceRail,
  tags: ["autodocs"],
  parameters: {
    docs: {
      description: {
        component:
          "Read-only Daggerheart reference rail that reuses the isolated card and sheet slices inside the interaction shell.",
      },
    },
  },
} satisfies Meta<typeof CharacterReferenceRail>;

export default meta;

type Story = StoryObj<typeof meta>;

export const WithSelectedSheet: Story = {
  args: characterReferenceRailFixtures,
};
