import type { PlayerHUDCharacterReference } from "../character-inspection-contract";

export type ParticipantPortraitStatus =
  | "idle"
  | "typing"
  | "ready"
  | "active"
  | "yielded"
  | "changes-requested";

export type ParticipantPortraitAIStatus = "thinking" | "failed";

export type ParticipantPortraitRailParticipant = {
  id: string;
  name: string;
  avatarUrl?: string;
  characters: PlayerHUDCharacterReference[];
  roleLabel?: string;
  status: ParticipantPortraitStatus;
  aiStatus?: ParticipantPortraitAIStatus;
  ownsGMAuthority?: boolean;
};

export type ParticipantPortraitRailProps = {
  participants: ParticipantPortraitRailParticipant[];
  viewerParticipantId: string;
  ariaLabel?: string;
  onParticipantInspect?: (participantId: string) => void;
};
