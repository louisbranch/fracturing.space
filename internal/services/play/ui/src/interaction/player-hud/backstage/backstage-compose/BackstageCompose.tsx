import { ChatCompose } from "../../chat/chat-compose/ChatCompose";
import type { BackstageComposeProps } from "./contract";

export function BackstageCompose({
  draft,
  disabled,
  onDraftChange,
  onSend,
}: BackstageComposeProps) {
  return (
    <ChatCompose
      draft={draft}
      disabled={disabled}
      onDraftChange={onDraftChange}
      onSend={onSend}
      ariaLabel="Backstage message input"
      placeholder={disabled ? "Waiting for OOC to open..." : "Add an OOC note, rules question, or clarification..."}
      sendLabel="Post"
    />
  );
}
