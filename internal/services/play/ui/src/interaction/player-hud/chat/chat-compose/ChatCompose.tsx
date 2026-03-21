import type { KeyboardEvent } from "react";
import type { ChatComposeProps } from "./contract";

// ChatCompose is the fixed-bottom compose bar for the side chat panel.
// It auto-expands vertically via CSS field-sizing and caps at max-h-32.
export function ChatCompose({
  draft,
  onDraftChange,
  onSend,
  disabled = false,
  ariaLabel = "Chat message input",
  placeholder = "Type a message...",
  sendLabel = "Send",
}: ChatComposeProps) {
  const canSend = !disabled && draft.trim().length > 0;

  function handleKeyDown(e: KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      if (canSend) onSend();
    }
  }

  return (
    <div className="flex shrink-0 gap-1.5 border-t border-base-300 p-1.5">
      <textarea
        aria-label={ariaLabel}
        className="textarea textarea-bordered max-h-28 min-h-0 flex-1 resize-none text-sm leading-snug"
        style={{ fieldSizing: "content" } as React.CSSProperties}
        rows={1}
        value={draft}
        disabled={disabled}
        onChange={(e) => onDraftChange(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
      />
      <button
        type="button"
        className="btn btn-sm btn-primary self-end"
        disabled={!canSend}
        onClick={onSend}
      >
        {sendLabel}
      </button>
    </div>
  );
}
