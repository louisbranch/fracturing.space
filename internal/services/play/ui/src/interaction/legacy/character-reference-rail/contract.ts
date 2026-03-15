import type { DaggerheartCharacterCardData } from "../../../systems/daggerheart/character-card/contract";
import type { DaggerheartCharacterSheetData } from "../../../systems/daggerheart/character-sheet/contract";

export type CharacterReferenceRailProps = {
  characters: DaggerheartCharacterCardData[];
  activeCharacterIds: string[];
  selectedCharacterId?: string;
  selectedSheet?: DaggerheartCharacterSheetData;
  onSelectCharacter?: (characterID: string) => void;
};
