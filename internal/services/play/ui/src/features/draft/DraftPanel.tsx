import { useState } from "react";
import type { TypingEvent } from "../../types";

type Props = {
  typing: TypingEvent[];
  onTypingChange: (active: boolean) => void;
};

export function DraftPanel({ typing, onTypingChange }: Props) {
  const [draft, setDraft] = useState("");

  return (
    <section className="play-panel">
      <div className="play-panel-body">
        <div className="play-panel-head">
          <div className="space-y-2">
            <p className="play-eyebrow">Composition</p>
            <h2 className="font-display text-3xl">Action draft</h2>
            <p className="play-prose">
              Draft scene actions or OOC coordination without committing them into the authoritative flow.
            </p>
          </div>
        </div>

        <div className="min-h-6 text-sm text-base-content/65">
          {typing.filter((event) => event.active).map((event) => (
            <span key={event.participant_id} className="block">
              {event.name || event.participant_id} is drafting...
            </span>
          ))}
        </div>

        <label className="form-control">
          <span className="label">
            <span className="label-text font-medium">Scratchpad</span>
          </span>
          <textarea
            className="textarea textarea-bordered min-h-40 w-full bg-base-100"
            value={draft}
            placeholder="Draft scene actions or OOC notes here..."
            rows={6}
            onChange={(event) => {
              const value = event.target.value;
              setDraft(value);
              onTypingChange(value.trim().length > 0);
            }}
          />
        </label>
      </div>
    </section>
  );
}
