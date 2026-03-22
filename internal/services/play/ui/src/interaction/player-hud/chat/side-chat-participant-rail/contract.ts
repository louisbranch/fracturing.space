import type { SideChatParticipant } from "../../shared/contract";
import type { OnStageAIStatus } from "../../on-stage/shared/contract";

export type SideChatParticipantRailProps = {
  participants: SideChatParticipant[];
  viewerParticipantId: string;
  aiOwnerParticipantId?: string;
  aiStatus?: OnStageAIStatus;
  onParticipantInspect?: (participantId: string) => void;
};
