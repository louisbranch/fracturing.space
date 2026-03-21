import type { BackstageParticipant } from "../shared/contract";

export type BackstageParticipantRailProps = {
  participants: BackstageParticipant[];
  viewerParticipantId: string;
  gmAuthorityParticipantId?: string;
  ariaLabel?: string;
};
