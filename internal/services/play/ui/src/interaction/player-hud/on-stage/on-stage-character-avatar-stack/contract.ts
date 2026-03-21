import type { OnStageCharacterSummary } from "../shared/contract";

export type OnStageCharacterAvatarStackProps = {
  characters: OnStageCharacterSummary[];
  ariaLabel?: string;
  onCharacterInspect?: (characterId: string) => void;
};
