import type { Meta, StoryObj } from "@storybook/react-vite";
import { CharacterCard, characterCardVariants } from "./CharacterCard";
import { CharacterCardStoryStage } from "./StoryStage";
import { characterCardFixtures } from "./fixtures";

const meta = {
  title: "Systems/Daggerheart/Character Card/Overview",
  component: CharacterCard,
  tags: ["autodocs"],
  args: {
    character: characterCardFixtures.full,
    variant: "full",
  },
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Reference view for the Character Card. The content hierarchy follows the web campaign character card and the web character detail Daggerheart summary, while the canvas shows each mode in a realistic single-card screen slot.",
      },
    },
  },
} satisfies Meta<typeof CharacterCard>;

export default meta;

type Story = StoryObj<typeof meta>;

export const OnScreenReferences: Story = {
  render: () => (
    <main className="preview-shell">
      <section className="preview-grid" aria-label="Character Card reference states">
        {characterCardVariants.map((variant) => (
          <section key={variant.id}>
            <CharacterCardStoryStage variant={variant.id}>
              <CharacterCard character={characterCardFixtures.full} variant={variant.id} />
            </CharacterCardStoryStage>
          </section>
        ))}
      </section>
    </main>
  ),
};
