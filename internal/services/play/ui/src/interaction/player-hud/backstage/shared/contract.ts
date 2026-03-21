import type {
  PlayerHUDCharacterInspectionCatalog,
  PlayerHUDCharacterReference,
} from "../../shared/character-inspection-contract";

export type BackstageMode = "dormant" | "open";
export type BackstageResumeState = "inactive" | "collecting-ready" | "waiting-on-gm";
export type BackstageParticipantRole = "player" | "gm";

export type BackstageParticipant = {
  id: string;
  name: string;
  role: BackstageParticipantRole;
  avatarUrl?: string;
  characters: PlayerHUDCharacterReference[];
  readyToResume: boolean;
  typing?: boolean;
};

export type BackstageMessage = {
  id: string;
  participantId: string;
  body: string;
  sentAt: string;
};

export type BackstageState = {
  mode: BackstageMode;
  sceneName?: string;
  pausedPromptText?: string;
  reason?: string;
  gmAuthorityParticipantId?: string;
  resumeState: BackstageResumeState;
  viewerParticipantId: string;
  participants: BackstageParticipant[];
  messages: BackstageMessage[];
  characterInspectionCatalog: PlayerHUDCharacterInspectionCatalog;
};
