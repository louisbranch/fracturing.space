import type { DaggerheartCharacterCardData } from "../../../systems/daggerheart/character-card/contract";
import type { DaggerheartCharacterSheetData } from "../../../systems/daggerheart/character-sheet/contract";

export type PlayerHUDCharacterReference = {
  id: string;
  name: string;
  avatarUrl?: string;
};

export type PlayerHUDCharacterInspection = {
  system: "daggerheart";
  card: DaggerheartCharacterCardData;
  sheet: DaggerheartCharacterSheetData;
};

export type PlayerHUDCharacterInspectionCatalog = Record<
  string,
  PlayerHUDCharacterInspection
>;
