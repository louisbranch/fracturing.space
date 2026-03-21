import type { KeyboardEvent } from "react";
import type { BackstageComposeProps } from "./contract";

export function BackstageCompose({
  draft,
  viewerReady,
  disabled,
  onDraftChange,
  onSend,
  onReadyToggle,
}: BackstageComposeProps) {
  const trimmedDraft = draft.trim();
  const canSend = !disabled && trimmedDraft.length > 0;
  const readyLabel = viewerReady ? "Clear Ready" : "Mark Ready";

  function handleKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault();
      if (canSend) {
        onSend();
      }
    }
  }

  return (
    <section aria-label="Backstage actions" className="border-t border-base-300/70 bg-base-300 px-3 py-3">
      <div className="flex flex-col gap-2 md:flex-row md:items-end">
        <textarea
          aria-label="Backstage message input"
          className="textarea textarea-bordered min-h-16 w-full flex-1 resize-y text-sm leading-snug"
          value={draft}
          disabled={disabled}
          onChange={(event) => onDraftChange(event.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Add an OOC note, rules question, or clarification..."
        />

        <div
          aria-label="Backstage action controls"
          className="flex shrink-0 justify-end md:self-stretch"
        >
          <div className="join join-vertical w-full md:w-36">
            <button
              type="button"
              className="btn btn-sm join-item btn-primary btn-soft w-full"
              disabled={!canSend}
              onClick={onSend}
            >
              Post
            </button>
            <button
              type="button"
              className={`btn btn-sm join-item w-full ${viewerReady ? "btn-warning btn-soft" : "btn-secondary btn-soft"}`}
              disabled={disabled}
              onClick={onReadyToggle}
            >
              {readyLabel}
            </button>
          </div>
        </div>
      </div>
    </section>
  );
}
