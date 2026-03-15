import { interactionCharacterFixtures, interactionFixtureCatalog } from "../shared/fixtures";

export const playInteractionShellFixtures = {
  playersOpen: {
    state: interactionFixtureCatalog.playersOpenMultiActor,
    references: interactionCharacterFixtures,
    showChatSidecar: true,
  },
  oocOpen: {
    state: interactionFixtureCatalog.oocOpenReadyToResume,
    references: interactionCharacterFixtures,
    showChatSidecar: true,
  },
  aiTurnFailed: {
    state: interactionFixtureCatalog.aiTurnFailed,
    references: interactionCharacterFixtures,
    showChatSidecar: true,
  },
};
