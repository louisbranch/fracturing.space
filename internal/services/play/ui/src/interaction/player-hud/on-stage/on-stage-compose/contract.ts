import type { OnStageMechanicsExtension, OnStageViewerControls } from "../shared/contract";

export type OnStageComposeProps = {
  draft: string;
  controls: OnStageViewerControls;
  mechanicsExtension?: OnStageMechanicsExtension;
  onDraftChange: (value: string) => void;
  onSubmit: () => void;
  onSubmitAndYield: () => void;
  onYield: () => void;
  onUnyield: () => void;
};
