import { ChatList } from "../../chat/chat-list/ChatList";
import type { BackstageOOCListProps } from "./contract";

export function BackstageOOCList({
  messages,
  participants,
  viewerParticipantId,
}: BackstageOOCListProps) {
  return (
    <ChatList
      messages={messages}
      participants={participants}
      viewerParticipantId={viewerParticipantId}
      ariaLabel="Backstage OOC messages"
      emptyLabel="No OOC messages yet"
    />
  );
}
