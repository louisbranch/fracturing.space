import { ChatCompose } from "../chat-compose/ChatCompose";
import { ChatList } from "../chat-list/ChatList";
import type { SideChatPanelProps } from "./contract";

// SideChatPanel composes ChatList and ChatCompose into a single panel.
// The center content region scrolls while the compose bar stays pinned.
export function SideChatPanel({ state, draft, onDraftChange, onSend }: SideChatPanelProps) {
  return (
    <section aria-label="Side chat" className="flex min-h-0 flex-1 flex-col">
      <div className="hud-panel-scroll-region">
        <ChatList
          messages={state.messages}
          participants={state.participants}
          viewerParticipantId={state.viewerParticipantId}
        />
      </div>
      <ChatCompose draft={draft} onDraftChange={onDraftChange} onSend={onSend} />
    </section>
  );
}
