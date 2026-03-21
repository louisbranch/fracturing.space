import { useEffect, useRef } from "react";
import type { SideChatParticipant } from "../../shared/contract";
import { ChatBubble } from "../chat-bubble/ChatBubble";
import type { ChatListProps } from "./contract";

// formatTime extracts hh:mm from an ISO timestamp.
function formatTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", hour12: false });
}

// ChatList renders grouped chat messages for the panel-owned scroll region.
// Consecutive messages from the same participant are visually grouped: the
// name appears on the first message and the avatar on the last in each run.
export function ChatList({
  messages,
  participants,
  viewerParticipantId,
  ariaLabel = "Side chat messages",
  emptyLabel = "No messages yet",
}: ChatListProps) {
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
        aria-label={ariaLabel}
        className="flex min-h-full flex-1 flex-col items-center justify-center rounded-box bg-base-200 px-3 py-3"
      >
        <span className="text-sm text-base-content/50">{emptyLabel}</span>
      </div>
    );
  }

  return (
    <div
      aria-label={ariaLabel}
      className="flex min-h-full flex-col rounded-box bg-base-200 px-1.5 py-1.5"
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
