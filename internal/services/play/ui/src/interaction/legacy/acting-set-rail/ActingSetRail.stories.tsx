import type { Meta, StoryObj } from "@storybook/react-vite";
import { ActingSetRail } from "./ActingSetRail";
import { actingSetRailFixtures } from "./fixtures";

const meta = {
  title: "Interaction/Legacy/Acting Set Rail",
  component: ActingSetRail,
  tags: ["autodocs"],
  parameters: {
    docs: {
      description: {
        component:
          "Character-focused view of the current acting set, independent from the player slot summaries.",
      },
    },
  },
} satisfies Meta<typeof ActingSetRail>;

export default meta;

type Story = StoryObj<typeof meta>;

export const MultiActor: Story = {
  args: {
    actingSet: actingSetRailFixtures.multiActor,
  },
};

export const SingleActor: Story = {
  args: {
    actingSet: actingSetRailFixtures.singleActor,
  },
};
