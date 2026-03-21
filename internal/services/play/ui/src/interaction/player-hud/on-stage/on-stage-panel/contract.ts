import type { OnStageState } from "../shared/contract";

export type OnStagePanelProps = {
  state: OnStageState;
  draft: string;
  onDraftChange: (value: string) => void;
  onSubmit: () => void;
  onSubmitAndYield: () => void;
  onYield: () => void;
  onUnyield: () => void;
};
