import type { OnStageParticipant } from "../shared/contract";

export type OnStageParticipantRailProps = {
  participants: OnStageParticipant[];
  viewerParticipantId: string;
  ariaLabel?: string;
};
