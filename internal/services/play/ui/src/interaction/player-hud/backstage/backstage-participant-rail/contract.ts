import type { BackstageParticipant } from "../shared/contract";
import type { OnStageAIStatus } from "../../on-stage/shared/contract";

export type BackstageParticipantRailProps = {
  participants: BackstageParticipant[];
  viewerParticipantId: string;
  gmAuthorityParticipantId?: string;
  aiOwnerParticipantId?: string;
  aiStatus?: OnStageAIStatus;
  ariaLabel?: string;
  onParticipantInspect?: (participantId: string) => void;
};
