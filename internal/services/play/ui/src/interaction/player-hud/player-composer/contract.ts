import type { PlayerComposerMode, PlayerComposerState } from "../shared/contract";

export type PlayerComposerActionHandlers = {
  onModeChange?: (mode: PlayerComposerMode) => void;
  onMinimizeChange?: (minimized: boolean) => void;
  onDraftChange?: (mode: PlayerComposerMode, draft: string) => void;
  onClearScratch?: () => void;
  onSceneYieldToggle?: () => void;
  onSceneSubmit?: () => void;
  onOOCPause?: () => void;
  onOOCResume?: () => void;
  onOOCSubmit?: () => void;
  onChatSubmit?: () => void;
};

export type PlayerComposerProps = {
  state: PlayerComposerState;
} & PlayerComposerActionHandlers;
