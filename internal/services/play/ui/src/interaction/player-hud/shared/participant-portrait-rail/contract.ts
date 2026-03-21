export type ParticipantPortraitStatus =
  | "idle"
  | "typing"
  | "ready"
  | "active"
  | "yielded"
  | "changes-requested";

export type ParticipantPortraitRailParticipant = {
  id: string;
  name: string;
  avatarUrl?: string;
  roleLabel?: string;
  status: ParticipantPortraitStatus;
  ownsGMAuthority?: boolean;
};

export type ParticipantPortraitRailProps = {
  participants: ParticipantPortraitRailParticipant[];
  viewerParticipantId: string;
  ariaLabel?: string;
};
