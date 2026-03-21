import type { SideChatParticipant } from "../../shared/contract";

export type SideChatParticipantRailProps = {
  participants: SideChatParticipant[];
  viewerParticipantId: string;
};
