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

export type OnStageCharacterSummary = {
  id: string;
  name: string;
  avatarUrl?: string;
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
  sceneName: string;
  sceneDescription?: string;
  gmOutputText?: string;
  frameText?: string;
  oocReason?: string;
  viewerParticipantId: string;
  actingParticipantIds: string[];
  actingCharacterNames: string[];
  gmAuthorityParticipantId?: string;
  participants: OnStageParticipant[];
  slots: OnStageSlot[];
  viewerControls: OnStageViewerControls;
  mechanicsExtension?: OnStageMechanicsExtension;
};
