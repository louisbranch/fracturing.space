import type { BackstageState } from "../backstage/shared/contract";
import type { HUDNavbarTab, SideChatState } from "../shared/contract";

export type PlayerHUDShellProps = {
  activeTab: HUDNavbarTab;
  onTabChange: (tab: HUDNavbarTab) => void;
  backstage: BackstageState;
  backstageDraft: string;
  onBackstageDraftChange: (value: string) => void;
  onBackstageSend: () => void;
  onBackstageReadyToggle: () => void;
  sideChat: SideChatState;
  sideChatDraft: string;
  onSideChatDraftChange: (value: string) => void;
  onSideChatSend: () => void;
};
