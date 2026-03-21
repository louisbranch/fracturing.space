import type { ChatBubbleProps } from "./contract";

// ChatBubble renders a single message bubble using DaisyUI's chat component.
// Consecutive messages from the same participant share a visual run: the name
// appears only on the first message and the avatar only on the last.
export function ChatBubble({
  body,
  time,
  alignment,
  showName,
  showAvatar,
  avatarUrl,
  avatarFallback,
}: ChatBubbleProps) {
  return (
    <div className={`chat chat-${alignment}`}>
      <div className="chat-image avatar">
        <div className={`w-10 rounded-full ${showAvatar ? "bg-base-300" : ""}`}>
          {showAvatar &&
            (avatarUrl ? (
              <img src={avatarUrl} alt={avatarFallback ?? ""} />
            ) : (
              <span className="flex h-full w-full items-center justify-center text-sm font-medium">
                {avatarFallback}
              </span>
            ))}
        </div>
      </div>
      {showName && <div className="chat-header">{showName}</div>}
      <div className="chat-bubble flex items-end gap-2">
        <span>{body}</span>
        <time className="shrink-0 text-[0.65rem] opacity-50">{time}</time>
      </div>
    </div>
  );
}
