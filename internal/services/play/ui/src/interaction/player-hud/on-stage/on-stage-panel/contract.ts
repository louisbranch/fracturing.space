import type { OnStageState } from "../shared/contract";

export type OnStagePanelProps = {
  state: OnStageState;
  draft: string;
  interactionTransitionActive?: boolean;
  onInteractionTransitionEnd?: () => void;
  onDraftChange: (value: string) => void;
  onSubmit: () => void;
  onSubmitAndYield: () => void;
  onYield: () => void;
  onUnyield: () => void;
  onCharacterInspect?: (participantId: string, characterId: string) => void;
};
