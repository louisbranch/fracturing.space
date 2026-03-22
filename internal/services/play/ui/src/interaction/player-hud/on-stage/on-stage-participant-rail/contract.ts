import type { OnStageAIStatus, OnStageParticipant } from "../shared/contract";

export type OnStageParticipantRailProps = {
  participants: OnStageParticipant[];
  viewerParticipantId: string;
  aiOwnerParticipantId?: string;
  aiStatus?: OnStageAIStatus;
  ariaLabel?: string;
  onParticipantInspect?: (participantId: string) => void;
};
