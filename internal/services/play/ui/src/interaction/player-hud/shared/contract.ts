import type { BackstageState } from "../backstage/shared/contract";
import type { OnStageState } from "../on-stage/shared/contract";

// HUDNavbarTab identifies the three top-level navigation surfaces in the player
// HUD.
export type HUDNavbarTab = "on-stage" | "backstage" | "side-chat";

// SideChatParticipant represents a user in the side chat conversation.
export type SideChatParticipant = {
  id: string;
  name: string;
  role: "player" | "gm";
  avatarUrl?: string;
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
};

// PlayerHUDState is the minimal top-level state for the player HUD shell.
export type PlayerHUDState = {
  activeTab: HUDNavbarTab;
  onStage: OnStageState;
  backstage: BackstageState;
  sideChat: SideChatState;
};
