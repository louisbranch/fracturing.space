import type { BackstageMessage, BackstageParticipant } from "../shared/contract";

export type BackstageOOCListProps = {
  messages: BackstageMessage[];
  participants: BackstageParticipant[];
  viewerParticipantId: string;
};
