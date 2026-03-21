import type { BackstageState } from "../backstage/shared/contract";
import type { OnStageState } from "../on-stage/shared/contract";
import type { HUDNavbarTab, SideChatState } from "../shared/contract";

export type PlayerHUDShellProps = {
  activeTab: HUDNavbarTab;
  onTabChange: (tab: HUDNavbarTab) => void;
  onStage: OnStageState;
  onStageDraft: string;
  onOnStageDraftChange: (value: string) => void;
  onOnStageSubmit: () => void;
  onOnStageSubmitAndYield: () => void;
  onOnStageYield: () => void;
  onOnStageUnyield: () => void;
  onCharacterInspect?: (participantId: string, characterId: string) => void;
  onParticipantInspect?: (participantId: string) => void;
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
