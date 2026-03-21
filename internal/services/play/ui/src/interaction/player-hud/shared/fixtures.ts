import { backstageFixtureCatalog, backstageParticipants } from "../backstage/shared/fixtures";
import {
  onStageCharacterCatalog,
  onStageFixtureCatalog,
  onStageParticipants,
} from "../on-stage/shared/fixtures";
import type { PlayerHUDState, SideChatMessage, SideChatParticipant, SideChatState } from "./contract";

export const sideChatParticipants: SideChatParticipant[] = [
  backstageParticipants[0],
  { ...backstageParticipants[1], typing: true },
  backstageParticipants[2],
];

export const sideChatMessages: SideChatMessage[] = [
  { id: "m1", participantId: "p-bryn", body: "Ready when you are.", sentAt: "2026-03-18T16:30:00Z" },
  { id: "m2", participantId: "p-bryn", body: "I'll take the left flank.", sentAt: "2026-03-18T16:30:15Z" },
  { id: "m3", participantId: "p-rhea", body: "Copy. Moving to the bridge.", sentAt: "2026-03-18T16:31:00Z" },
  { id: "m4", participantId: "p-guide", body: "Quick heads-up: I'm adding a weather complication next round.", sentAt: "2026-03-18T16:32:00Z" },
  { id: "m5", participantId: "p-rhea", body: "Sounds good!", sentAt: "2026-03-18T16:32:30Z" },
  { id: "m6", participantId: "p-rhea", body: "Should we prep anything?", sentAt: "2026-03-18T16:32:45Z" },
];

export const sideChatState: SideChatState = {
  viewerParticipantId: "p-rhea",
  participants: sideChatParticipants,
  messages: sideChatMessages,
};

export const emptySideChatState: SideChatState = {
  viewerParticipantId: "p-rhea",
  participants: sideChatParticipants,
  messages: [],
};

export const playerHUDFixtureCatalog: Record<
  "onStage" | "backstage" | "sideChat",
  PlayerHUDState
> = {
  onStage: {
    activeTab: "on-stage",
    onStage: onStageFixtureCatalog.viewerPosted,
    backstage: backstageFixtureCatalog.dormant,
    sideChat: sideChatState,
  },
  backstage: {
    activeTab: "backstage",
    onStage: onStageFixtureCatalog.waitingOnGM,
    backstage: backstageFixtureCatalog.openDiscussion,
    sideChat: sideChatState,
  },
  sideChat: {
    activeTab: "side-chat",
    onStage: onStageFixtureCatalog.aiThinking,
    backstage: backstageFixtureCatalog.waitingOnGM,
    sideChat: sideChatState,
  },
};

export {
  backstageFixtureCatalog,
  backstageParticipants,
  onStageCharacterCatalog,
  onStageFixtureCatalog,
  onStageParticipants,
};
