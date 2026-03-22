import type { BackstageState } from "../backstage/shared/contract";
import type { OnStageState } from "../on-stage/shared/contract";
import type {
  PlayerHUDCharacterInspectionCatalog,
  PlayerHUDCharacterReference,
} from "./character-inspection-contract";

// HUDNavbarTab identifies the three top-level navigation surfaces in the player
// HUD.
export type HUDNavbarTab = "on-stage" | "backstage" | "side-chat" | "ai-debug";

// HUDConnectionState identifies the transport health state shown in the shared
// Player HUD navbar badge.
export type HUDConnectionState = "connected" | "reconnecting" | "disconnected";

export type PlayerHUDCharacterController = {
  participantId: string;
  participantName: string;
  isViewer: boolean;
  characters: PlayerHUDCharacterReference[];
};

export type PlayerHUDCampaignNavigation = {
  returnHref: string;
  characterControllers: PlayerHUDCharacterController[];
  characterInspectionCatalog: PlayerHUDCharacterInspectionCatalog;
};

// SideChatParticipant represents a user in the side chat conversation.
export type SideChatParticipant = {
  id: string;
  name: string;
  role: "player" | "gm";
  avatarUrl?: string;
  characters: PlayerHUDCharacterReference[];
  typing?: boolean;
};

// SideChatMessage is a single message in the side chat.
export type SideChatMessage = {
  id: string;
  participantId: string;
  body: string;
  sentAt: string; // ISO timestamp, rendered as hh:mm
};

// SideChatState holds the full state for the side chat panel.
export type SideChatState = {
  viewerParticipantId: string;
  participants: SideChatParticipant[];
  messages: SideChatMessage[];
  characterInspectionCatalog: PlayerHUDCharacterInspectionCatalog;
};

// PlayerHUDState is the minimal top-level state for the player HUD shell.
export type PlayerHUDState = {
  activeTab: HUDNavbarTab;
  connectionState: HUDConnectionState;
  campaignNavigation: PlayerHUDCampaignNavigation;
  onStage: OnStageState;
  backstage: BackstageState;
  sideChat: SideChatState;
};
