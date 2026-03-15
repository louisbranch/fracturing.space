import type { CharacterReferenceFixtures, PlayInteractionFixtureData } from "../shared/contract";

export type PlayInteractionShellProps = {
  state: PlayInteractionFixtureData;
  references: CharacterReferenceFixtures;
  showChatSidecar?: boolean;
};
