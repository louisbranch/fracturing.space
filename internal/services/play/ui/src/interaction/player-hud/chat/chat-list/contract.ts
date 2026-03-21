import type { SideChatMessage, SideChatParticipant } from "../../shared/contract";

export type ChatListProps = {
  messages: SideChatMessage[];
  participants: SideChatParticipant[];
  viewerParticipantId: string;
  ariaLabel?: string;
  emptyLabel?: string;
};
