import type { ReactNode } from "react";
import type { CharacterCardVariant } from "./contract";

const stageCopy: Record<CharacterCardVariant, { cardClass: string }> = {
  portrait: {
    cardClass: "story-stage-card story-stage-card-portrait",
  },
  basic: {
    cardClass: "story-stage-card story-stage-card-basic",
  },
};

// CharacterCardStoryStage keeps Storybook canvases focused on realistic screen
// slots instead of mixing card content with bespoke preview navigation.
export function CharacterCardStoryStage(input: {
  variant: CharacterCardVariant;
  children: ReactNode;
}) {
  const metadata = stageCopy[input.variant];

  return <div className={metadata.cardClass}>{input.children}</div>;
}
