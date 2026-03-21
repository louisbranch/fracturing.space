import type { OnStageCharacterSummary } from "../shared/contract";

export type OnStageSceneCardProps = {
  sceneName: string;
  sceneDescription?: string;
  sceneCharacters: OnStageCharacterSummary[];
  resolvedInteractionCount: number;
  expanded: boolean;
  onToggle: () => void;
  onCharacterInspect?: (characterId: string) => void;
};
