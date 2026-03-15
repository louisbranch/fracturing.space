import type { ReactNode } from "react";

// CharacterSheetStoryStage keeps Storybook canvases focused on a realistic
// screen slot for the full-width character sheet layout.
export function CharacterSheetStoryStage(input: { children: ReactNode }) {
  return <div className="story-stage-card story-stage-card-sheet">{input.children}</div>;
}
