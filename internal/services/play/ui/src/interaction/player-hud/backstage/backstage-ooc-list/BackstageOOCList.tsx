import { useEffect, useRef } from "react";
import type { BackstageParticipant } from "../shared/contract";
import type { BackstageOOCListProps } from "./contract";

function formatTime(iso: string): string {
  const date = new Date(iso);
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", hour12: false });
}

export function BackstageOOCList({
  messages,
  participants,
  viewerParticipantId,
}: BackstageOOCListProps) {
  const bottomRef = useRef<HTMLDivElement>(null);
  const participantMap = new Map<string, BackstageParticipant>(
    participants.map((participant) => [participant.id, participant]),
  );

  useEffect(() => {
    bottomRef.current?.scrollIntoView?.({ behavior: "instant" });
  }, [messages.length]);

  if (messages.length === 0) {
    return (
      <div
        aria-label="Backstage OOC messages"
        className="flex min-h-0 flex-1 flex-col items-center justify-center overflow-y-auto"
      >
        <span className="text-sm text-base-content/50">No OOC messages yet</span>
      </div>
    );
  }

  return (
    <div
      aria-label="Backstage OOC messages"
      className="flex min-h-0 flex-1 flex-col overflow-y-auto px-2 py-2"
    >
      {messages.map((message, index) => {
        const previous = messages[index - 1];
        const next = messages[index + 1];
        const participant = participantMap.get(message.participantId);
        const name = participant?.name ?? "Unknown";
        const isViewer = message.participantId === viewerParticipantId;
        const isFirstInRun = !previous || previous.participantId !== message.participantId;
        const isLastInRun = !next || next.participantId !== message.participantId;

        return (
          <div key={message.id} className={`chat chat-${isViewer ? "end" : "start"}`}>
            <div className="chat-image avatar">
              <div className={`w-10 rounded-full ${isLastInRun ? "bg-base-300" : ""}`}>
                {isLastInRun ? (
                  participant?.avatarUrl ? (
                    <img src={participant.avatarUrl} alt={name.charAt(0)} />
                  ) : (
                    <span className="flex h-full w-full items-center justify-center text-sm font-medium">
                      {name.charAt(0)}
                    </span>
                  )
                ) : null}
              </div>
            </div>
            {isFirstInRun ? <div className="chat-header">{name}</div> : null}
            <div className="chat-bubble flex items-end gap-2">
              <span>{message.body}</span>
              <time className="shrink-0 text-[0.65rem] opacity-50">{formatTime(message.sentAt)}</time>
            </div>
          </div>
        );
      })}
      <div ref={bottomRef} />
    </div>
  );
}
