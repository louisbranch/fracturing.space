import type { SideChatState } from "../../shared/contract";

export type SideChatPanelProps = {
  state: SideChatState;
  draft: string;
  onDraftChange: (value: string) => void;
  onSend: () => void;
};
