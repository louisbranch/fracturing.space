import { ChatCompose } from "../chat-compose/ChatCompose";
import { ChatList } from "../chat-list/ChatList";
import type { SideChatPanelProps } from "./contract";

// SideChatPanel composes ChatList and ChatCompose into a single panel.
// The list scrolls internally while the compose bar stays pinned at the bottom.
export function SideChatPanel({ state, draft, onDraftChange, onSend }: SideChatPanelProps) {
  return (
    <section aria-label="Side chat" className="flex min-h-0 flex-1 flex-col">
      <ChatList
        messages={state.messages}
        participants={state.participants}
        viewerParticipantId={state.viewerParticipantId}
      />
      <ChatCompose draft={draft} onDraftChange={onDraftChange} onSend={onSend} />
    </section>
  );
}
