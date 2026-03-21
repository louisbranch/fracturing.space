import { useEffect, useRef } from "react";
import type { SideChatParticipant } from "../../shared/contract";
import { ChatBubble } from "../chat-bubble/ChatBubble";
import type { ChatListProps } from "./contract";

// formatTime extracts hh:mm from an ISO timestamp.
function formatTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", hour12: false });
}

// ChatList renders a scrollable list of grouped chat messages. Consecutive
// messages from the same participant are visually grouped: the name appears on
// the first message and the avatar on the last in each run.
export function ChatList({ messages, participants, viewerParticipantId }: ChatListProps) {
  const bottomRef = useRef<HTMLDivElement>(null);
  const participantMap = new Map<string, SideChatParticipant>(
    participants.map((p) => [p.id, p]),
  );

  useEffect(() => {
    bottomRef.current?.scrollIntoView?.({ behavior: "instant" });
  }, [messages.length]);

  if (messages.length === 0) {
    return (
      <div
        aria-label="Side chat messages"
        className="flex min-h-0 flex-1 flex-col items-center justify-center overflow-y-auto"
      >
        <span className="text-sm text-base-content/50">No messages yet</span>
      </div>
    );
  }

  return (
    <div
      aria-label="Side chat messages"
      className="flex min-h-0 flex-1 flex-col overflow-y-auto px-2 py-2"
    >
      {messages.map((msg, i) => {
        const prev = messages[i - 1];
        const next = messages[i + 1];
        const participant = participantMap.get(msg.participantId);
        const name = participant?.name ?? "Unknown";
        const isViewer = msg.participantId === viewerParticipantId;

        const isFirstInRun = !prev || prev.participantId !== msg.participantId;
        const isLastInRun = !next || next.participantId !== msg.participantId;

        return (
          <ChatBubble
            key={msg.id}
            body={msg.body}
            time={formatTime(msg.sentAt)}
            alignment={isViewer ? "end" : "start"}
            showName={isFirstInRun ? name : undefined}
            showAvatar={isLastInRun}
            avatarUrl={participant?.avatarUrl}
            avatarFallback={name.charAt(0)}
          />
        );
      })}
      <div ref={bottomRef} />
    </div>
  );
}
