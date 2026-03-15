import type { CharacterCardPortrait, DaggerheartTrackValue } from "../character-card/contract";

// DaggerheartTrait models a single trait with its associated skill list for
// the full character sheet layout.
export type DaggerheartTrait = {
  name: string;
  abbreviation: string;
  value: number;
  skills: string[];
};

// DaggerheartWeapon models a weapon entry on the character sheet.
export type DaggerheartWeapon = {
  name: string;
  trait?: string;
  range?: string;
  damageDice?: string;
  damageType?: string;
  feature?: string;
};

// DaggerheartArmor models equipped armor on the character sheet.
export type DaggerheartArmor = {
  name: string;
  baseThresholds?: number;
  baseScore?: number;
  feature?: string;
};

// DaggerheartGold models the three-tier gold inventory.
export type DaggerheartGold = {
  handfuls: number;
  bags: number;
  chests: number;
};

// DaggerheartExperience keeps freeform experience rows with optional numeric
// modifiers for the sheet display.
export type DaggerheartExperience = {
  name: string;
  modifier?: number;
};

// DaggerheartDomainCard models a domain card reference on the sheet.
export type DaggerheartDomainCard = {
  name: string;
  domain?: string;
};

// DaggerheartCharacterSheetData is the full read-only data contract for the
// character sheet component — a superset of the card contract with structured
// equipment, traits, and narrative fields.
export type DaggerheartCharacterSheetData = {
  id: string;
  name: string;
  portrait: CharacterCardPortrait;
  pronouns?: string;
  level?: number;
  className?: string;
  subclassName?: string;
  ancestryName?: string;
  communityName?: string;
  proficiency?: number;

  traits?: DaggerheartTrait[];

  hp?: DaggerheartTrackValue;
  stress?: DaggerheartTrackValue;
  majorThreshold?: number;
  severeThreshold?: number;

  evasion?: number;
  armor?: DaggerheartTrackValue;

  hope?: DaggerheartTrackValue;
  hopeFeature?: string;

  classFeature?: string;

  primaryWeapon?: DaggerheartWeapon;
  secondaryWeapon?: DaggerheartWeapon;
  activeArmor?: DaggerheartArmor;

  experiences?: DaggerheartExperience[];
  domainCards?: DaggerheartDomainCard[];

  gold?: DaggerheartGold;

  description?: string;
  background?: string;
  connections?: string;

  // Runtime state — digital advantage over paper.
  lifeState?: "alive" | "unconscious" | "blaze_of_glory" | "dead";
  conditions?: string[];

  // Identity metadata.
  kind?: string;
  controller?: string;
};

// CharacterSheetProps keeps the public component seam narrow and prop-driven.
export type CharacterSheetProps = {
  character: DaggerheartCharacterSheetData;
};
