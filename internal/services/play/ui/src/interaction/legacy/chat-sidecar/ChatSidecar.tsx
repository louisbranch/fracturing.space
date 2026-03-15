import type { ChatSidecarProps } from "./contract";

// ChatSidecar keeps human transcript traffic visually separate from
// authoritative interaction state.
export function ChatSidecar({ messages }: ChatSidecarProps) {
  return (
    <aside className="preview-panel" aria-label="Session chat">
      <div className="preview-panel-body gap-4">
        <div>
          <span className="preview-kicker">Chat Sidecar</span>
          <h2 className="font-display text-2xl text-base-content">Human Transcript</h2>
          <p className="mt-2 text-sm leading-6 text-base-content/72">
            Useful table chatter and context, but not the source of gameplay authority.
          </p>
        </div>

        <div className="space-y-3">
          {messages.map((message) => (
            <article key={message.messageId} className="rounded-box border border-base-300/70 bg-base-100/65 px-4 py-3">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <strong className="text-sm text-base-content">{message.actorName}</strong>
                <span className="text-xs uppercase tracking-[0.16em] text-base-content/40">{message.sentAt}</span>
              </div>
              <p className="mt-2 text-sm leading-6 text-base-content/80">{message.body}</p>
            </article>
          ))}
        </div>
      </div>
    </aside>
  );
}
