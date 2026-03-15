import type { PlayPlayerSlotData } from "../shared/contract";

export type GMReviewPanelProps = {
  slots: PlayPlayerSlotData[];
  onAcceptPhase?: () => void;
  onRequestRevisions?: () => void;
  onEndPhase?: () => void;
};
