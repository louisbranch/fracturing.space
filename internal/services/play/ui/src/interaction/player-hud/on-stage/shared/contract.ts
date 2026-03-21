import type {
  PlayerHUDCharacterInspectionCatalog,
  PlayerHUDCharacterReference,
} from "../../shared/character-inspection-contract";

export type OnStageMode =
  | "waiting-on-gm"
  | "acting"
  | "yielded-waiting"
  | "changes-requested"
  | "ooc-blocked";

export type OnStageAIStatus = "idle" | "queued" | "running" | "failed";
export type OnStageParticipantRole = "player" | "gm";
export type OnStageParticipantRailStatus =
  | "waiting"
  | "active"
  | "yielded"
  | "changes-requested";
export type OnStageSlotReviewState =
  | "open"
  | "under-review"
  | "accepted"
  | "changes-requested";

export type OnStageCharacterSummary = PlayerHUDCharacterReference;

export type OnStageGMBeatType =
  | "fiction"
  | "prompt"
  | "resolution"
  | "consequence"
  | "guidance";

export type OnStageGMInteractionIllustration = {
  imageUrl: string;
  alt: string;
  caption?: string;
  sizeHint?: "compact" | "wide";
};

export type OnStageGMBeat = {
  id: string;
  type: OnStageGMBeatType;
  text: string;
};

export type OnStageGMInteraction = {
  id: string;
  title: string;
  characterIds: string[];
  illustration?: OnStageGMInteractionIllustration;
  beats: OnStageGMBeat[];
};

export type OnStageScene = {
  id: string;
  name: string;
  description?: string;
  characters: OnStageCharacterSummary[];
  resolvedInteractionCount: number;
};

export type OnStageParticipant = {
  id: string;
  name: string;
  role: OnStageParticipantRole;
  avatarUrl?: string;
  characters: OnStageCharacterSummary[];
  railStatus: OnStageParticipantRailStatus;
  ownsGMAuthority?: boolean;
};

export type OnStageSlot = {
  id: string;
  participantId: string;
  characters: OnStageCharacterSummary[];
  body?: string;
  updatedAt?: string;
  yielded: boolean;
  reviewState: OnStageSlotReviewState;
  reviewReason?: string;
};

export type OnStageViewerControls = {
  canSubmit: boolean;
  canSubmitAndYield: boolean;
  canYield: boolean;
  canUnyield: boolean;
  disabledReason?: string;
};

export type OnStageMechanicsExtension = {
  label: string;
  description: string;
};

export type OnStageState = {
  mode: OnStageMode;
  aiStatus: OnStageAIStatus;
  scene: OnStageScene;
  currentInteraction?: OnStageGMInteraction;
  interactionHistory: OnStageGMInteraction[];
  oocReason?: string;
  viewerParticipantId: string;
  actingParticipantIds: string[];
  gmAuthorityParticipantId?: string;
  participants: OnStageParticipant[];
  slots: OnStageSlot[];
  characterInspectionCatalog: PlayerHUDCharacterInspectionCatalog;
  viewerControls: OnStageViewerControls;
  mechanicsExtension?: OnStageMechanicsExtension;
};
