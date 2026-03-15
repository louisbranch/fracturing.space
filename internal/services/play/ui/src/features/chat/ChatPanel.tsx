import { useState } from "react";
import type { PlayChatMessage, TypingEvent } from "../../protocol";
import { formatClock } from "../../utils";

type Props = {
  connected: boolean;
  loadingHistory: boolean;
  messages: PlayChatMessage[];
  typing: TypingEvent[];
  onLoadOlder: () => Promise<void>;
  onSend: (body: string) => void;
  onTypingChange: (active: boolean) => void;
};

export function ChatPanel({
  connected,
  loadingHistory,
  messages,
  typing,
  onLoadOlder,
  onSend,
  onTypingChange,
}: Props) {
  const [draft, setDraft] = useState("");

  return (
    <section className="play-panel">
      <div className="play-panel-body">
        <div className="play-panel-head">
          <div className="space-y-2">
            <p className="play-eyebrow">Transcript</p>
            <h2 className="font-display text-3xl">Table chat</h2>
            <p className="play-prose">
              Human chat remains live even while the interaction state shifts between scenes.
            </p>
          </div>
          <span className={`badge badge-outline ${connected ? "badge-success" : "badge-error"}`}>
            {connected ? "Live" : "Offline"}
          </span>
        </div>

        <div className="flex justify-end">
          <button
            className="btn btn-ghost btn-sm"
            type="button"
            onClick={() => void onLoadOlder()}
          >
            {loadingHistory ? "Loading..." : "Load older"}
          </button>
        </div>

        <ol className="play-scroll-list">
          {messages.map((message) => (
            <li key={message.message_id}>
              <article className="chat chat-start">
                <div className="chat-header mb-2 flex items-center gap-2 text-sm text-base-content/60">
                  <strong className="text-base-content">
                    {message.actor.name || message.actor.participant_id}
                  </strong>
                  <time>{formatClock(message.sent_at)}</time>
                </div>
                <div className="chat-bubble max-w-full whitespace-pre-wrap bg-base-100 text-base-content shadow">
                  {message.body}
                </div>
              </article>
            </li>
          ))}
          {messages.length === 0 ? (
            <li className="rounded-box border border-dashed border-base-300 bg-base-100/40 px-4 py-6 text-sm text-base-content/60">
              No human chat messages yet.
            </li>
          ) : null}
        </ol>

        <div className="min-h-6 text-sm text-base-content/65">
          {typing.filter((event) => event.active).map((event) => (
            <span key={event.participant_id} className="block">
              {event.name || event.participant_id} is typing...
            </span>
          ))}
        </div>

        <form
          className="flex flex-col gap-3"
          onSubmit={(event) => {
            event.preventDefault();
            const body = draft.trim();
            if (!body) {
              return;
            }
            onSend(body);
            setDraft("");
            onTypingChange(false);
          }}
        >
          <label className="form-control">
            <span className="label">
              <span className="label-text font-medium">Message</span>
            </span>
            <textarea
              className="textarea textarea-bordered min-h-28 w-full bg-base-100"
              value={draft}
              placeholder="Send a human chat message..."
              onChange={(event) => {
                const value = event.target.value;
                setDraft(value);
                onTypingChange(value.trim().length > 0);
              }}
              rows={3}
            />
          </label>
          <div className="flex justify-end">
            <button className="btn btn-primary" type="submit">
              Send
            </button>
          </div>
        </form>
      </div>
    </section>
  );
}
