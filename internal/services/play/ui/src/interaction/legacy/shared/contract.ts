import type { DaggerheartCharacterCardData } from "../../../systems/daggerheart/character-card/contract";
import type { DaggerheartCharacterSheetData } from "../../../systems/daggerheart/character-sheet/contract";

// PlayViewerRole keeps the workflow-oriented viewer identity narrow for
// Storybook-first interaction slices.
export type PlayViewerRole = "gm" | "player";

// PlayPhaseStatus mirrors the durable interaction-owned phase states without
// leaking raw transport enums into components.
export type PlayPhaseStatus = "gm" | "players" | "gm_review";

// PlayPhaseData captures the current authority state shown across scene and
// shell components.
export type PlayPhaseData = {
  status: PlayPhaseStatus;
  gmAuthorityName: string;
  oocOpen?: boolean;
  summary?: string;
};

// PlaySceneData captures the narrative scene anchor for the active beat.
export type PlaySceneData = {
  name: string;
  description?: string;
  gmOutputText?: string;
  gmOutputAuthor?: string;
  frameText?: string;
};

// PlayActingCharacterData captures the acting set without depending on runtime
// ownership lookups.
export type PlayActingCharacterData = {
  id: string;
  name: string;
  participantName: string;
  spotlight?: boolean;
};

// PlayPlayerSlotReviewStatus mirrors the durable review lifecycle used by the
// player slot board.
export type PlayPlayerSlotReviewStatus =
  | "open"
  | "under_review"
  | "accepted"
  | "changes_requested";

// PlayPlayerSlotData is the Storybook-facing slot contract used across open and
// review states.
export type PlayPlayerSlotData = {
  participantId: string;
  participantName: string;
  summaryText?: string;
  characterNames: string[];
  yielded: boolean;
  reviewStatus?: PlayPlayerSlotReviewStatus;
  reviewReason?: string;
  isViewer?: boolean;
};

// PlayOOCPostData keeps OOC rendering explicit and transport-independent.
export type PlayOOCPostData = {
  postId: string;
  participantName: string;
  body: string;
  emphasis?: "gm" | "player";
};

// PlayOOCData captures the pause overlay, including ready-to-resume state.
export type PlayOOCData = {
  reason: string;
  posts: PlayOOCPostData[];
  readyParticipantNames: string[];
  viewerReady?: boolean;
};

// PlayAITurnStatus tracks the cross-service AI turn lifecycle for preview.
export type PlayAITurnStatus = "idle" | "queued" | "running" | "failed";

// PlayAITurnData captures the durable UI-facing AI turn surface.
export type PlayAITurnData = {
  status: PlayAITurnStatus;
  ownerName?: string;
  sourceLabel?: string;
  lastError?: string;
  canRetry?: boolean;
};

// PlayChatMessageData keeps the chat sidecar preview contract explicit.
export type PlayChatMessageData = {
  messageId: string;
  actorName: string;
  body: string;
  sentAt: string;
  emphasis?: "gm" | "player" | "system";
};

// PlayInteractionFixtureData is the composition-level fixture contract used by
// the Storybook shell and per-slice fixtures.
export type PlayInteractionFixtureData = {
  title: string;
  campaignName: string;
  sessionName: string;
  systemName: string;
  viewerName: string;
  viewerRole: PlayViewerRole;
  phase: PlayPhaseData;
  scene?: PlaySceneData;
  actingSet: PlayActingCharacterData[];
  slots: PlayPlayerSlotData[];
  ooc?: PlayOOCData;
  aiTurn?: PlayAITurnData;
  chat: PlayChatMessageData[];
};

// CharacterReferenceFixtures groups the read-only Daggerheart reference assets
// used by the interaction shell stories.
export type CharacterReferenceFixtures = {
  characters: DaggerheartCharacterCardData[];
  selectedSheet: DaggerheartCharacterSheetData;
  selectedCharacterId: string;
  activeCharacterIds: string[];
};
