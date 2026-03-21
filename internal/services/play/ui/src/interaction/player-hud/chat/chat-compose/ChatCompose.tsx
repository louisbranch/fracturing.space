import type { KeyboardEvent } from "react";
import type { ChatComposeProps } from "./contract";

// ChatCompose is the fixed-bottom compose bar for the side chat panel.
// It auto-expands vertically via CSS field-sizing and caps at max-h-32.
export function ChatCompose({ draft, onDraftChange, onSend }: ChatComposeProps) {
  const canSend = draft.trim().length > 0;

  function handleKeyDown(e: KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      if (canSend) onSend();
    }
  }

  return (
    <div className="flex shrink-0 gap-2 border-t border-base-300 p-2">
      <textarea
        aria-label="Chat message input"
        className="textarea textarea-bordered max-h-32 min-h-0 flex-1 resize-none leading-snug"
        style={{ fieldSizing: "content" } as React.CSSProperties}
        rows={1}
        value={draft}
        onChange={(e) => onDraftChange(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder="Type a message..."
      />
      <button
        type="button"
        className="btn btn-primary self-end"
        disabled={!canSend}
        onClick={onSend}
      >
        Send
      </button>
    </div>
  );
}
