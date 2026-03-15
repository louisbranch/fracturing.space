import type { PlayOOCData, PlayPhaseData } from "../shared/contract";

export type OOCOverlayPanelProps = {
  phase: PlayPhaseData;
  ooc: PlayOOCData;
  onResume?: () => void;
};
