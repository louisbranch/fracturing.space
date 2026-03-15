// CharacterCardVariant encodes the supported MVP density levels so preview,
// tests, and future callers share one small state surface.
export type CharacterCardVariant = "portrait" | "basic";

// CharacterCardPortrait keeps accessibility explicit even when preview fixtures
// intentionally omit the actual portrait source.
export type CharacterCardPortrait = {
  alt: string;
  src?: string;
  width?: number;
  height?: number;
};

// CharacterCardIdentity keeps the non-Daggerheart fields aligned with the web
// campaign character card rather than a generic profile-card vocabulary.
export type CharacterCardIdentity = {
  kind?: string;
  controller?: string;
  pronouns?: string;
  aliases?: string[];
};

// DaggerheartTrackValue models bounded Daggerheart resource-style values.
export type DaggerheartTrackValue = {
  current: number;
  max: number;
};

// DaggerheartCharacterSummary keeps the lightweight game-specific fields that
// appear on the web campaign character card.
export type DaggerheartCharacterSummary = {
  level?: number;
  className?: string;
  subclassName?: string;
  ancestryName?: string;
  communityName?: string;
  hp?: DaggerheartTrackValue;
  stress?: DaggerheartTrackValue;
  evasion?: number;
  armor?: DaggerheartTrackValue;
  hope?: DaggerheartTrackValue;
  feature?: string;
};

// DaggerheartCharacterTraits mirrors the compact trait values rendered in the
// web character detail summary.
export type DaggerheartCharacterTraits = {
  agility?: string;
  strength?: string;
  finesse?: string;
  instinct?: string;
  presence?: string;
  knowledge?: string;
};

// DaggerheartCharacterCardData is the stable external input contract that
// future runtime adapters or alternate card implementations must preserve.
export type DaggerheartCharacterCardData = {
  id: string;
  name: string;
  portrait: CharacterCardPortrait;
  identity?: CharacterCardIdentity;
  daggerheart?: {
    summary?: DaggerheartCharacterSummary;
    traits?: DaggerheartCharacterTraits;
  };
};

// CharacterCardProps keeps the public component seam narrow and prop-driven.
export type CharacterCardProps = {
  character: DaggerheartCharacterCardData;
  variant: CharacterCardVariant;
};
