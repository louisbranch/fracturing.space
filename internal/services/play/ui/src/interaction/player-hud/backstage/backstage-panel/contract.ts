import type { BackstageState } from "../shared/contract";

export type BackstagePanelProps = {
  state: BackstageState;
  draft: string;
  onDraftChange: (value: string) => void;
  onSend: () => void;
  onReadyToggle: () => void;
};
