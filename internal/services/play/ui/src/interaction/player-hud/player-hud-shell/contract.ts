import type { HUDNavbarTab, SideChatState } from "../shared/contract";

export type PlayerHUDShellProps = {
  activeTab: HUDNavbarTab;
  onTabChange: (tab: HUDNavbarTab) => void;
  sideChat: SideChatState;
  sideChatDraft: string;
  onSideChatDraftChange: (value: string) => void;
  onSideChatSend: () => void;
};
