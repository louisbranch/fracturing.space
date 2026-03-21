import type { KeyboardEvent } from "react";
import type { OnStageComposeProps } from "./contract";

export function OnStageCompose({
  draft,
  controls,
  mechanicsExtension: _mechanicsExtension,
  onDraftChange,
  onSubmit,
  onSubmitAndYield,
  onYield,
  onUnyield,
}: OnStageComposeProps) {
  const trimmedDraft = draft.trim();
  const canSend = trimmedDraft.length > 0;
  const inputDisabled =
    !controls.canSubmit &&
    !controls.canSubmitAndYield &&
    !controls.canYield &&
    !controls.canUnyield;
  const placeholder = "Commit the next action for your character...";

  function handleKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key === "Enter" && (event.metaKey || event.ctrlKey) && controls.canSubmit && canSend) {
      event.preventDefault();
      onSubmit();
    }
  }

  return (
    <section aria-label="On-stage actions" className="border-t border-base-300/70 bg-base-300 px-3 py-3">
      <div className="space-y-2">
        <div className="flex flex-col gap-2 md:flex-row md:items-end">
          <textarea
            aria-label="On-stage action input"
            className="textarea textarea-bordered min-h-24 w-full flex-1 resize-y text-sm leading-snug"
            value={draft}
            disabled={inputDisabled}
            onChange={(event) => onDraftChange(event.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={placeholder}
          />

          <div
            aria-label="On-stage action controls"
            className="flex shrink-0 justify-end md:self-stretch"
          >
            <div className="join join-vertical w-full md:w-44">
            {controls.canSubmit ? (
              <button
                type="button"
                className="btn btn-sm join-item btn-primary btn-soft w-full"
                disabled={!canSend}
                onClick={onSubmit}
              >
                Submit
              </button>
            ) : null}

            {controls.canSubmitAndYield ? (
              <button
                type="button"
                className="btn btn-sm join-item btn-primary btn-soft w-full"
                disabled={!canSend}
                onClick={onSubmitAndYield}
              >
                Submit &amp; Yield
              </button>
            ) : null}

            {controls.canYield ? (
              <button
                type="button"
                className="btn btn-sm join-item btn-secondary btn-soft w-full"
                onClick={onYield}
              >
                Yield
              </button>
            ) : null}

            {controls.canUnyield ? (
              <button
                type="button"
                className="btn btn-sm join-item btn-warning btn-soft w-full"
                onClick={onUnyield}
              >
                Unyield
              </button>
            ) : null}
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
