import type { DaggerheartCharacterCardData } from "./contract";
import { characterAvatarPreviewAssets } from "../../../storybook/preview-assets/fixtures";

// CharacterCardFixtures keeps the known mock-data set explicit for preview and
// test reuse across the isolated workflow.
type CharacterCardFixtures = Record<"full" | "minimal" | "partial", DaggerheartCharacterCardData>;

// The textual fixture values below are copied from current web references:
// - campaign character identity and Daggerheart summary from
//   `internal/services/web/modules/campaigns/render/detail_test.go`
// - Daggerheart full-info content from
//   `internal/services/web/modules/campaigns/render/character_creation_test.go`
//   and `internal/services/web/modules/campaigns/workflow/daggerheart/view_test.go`
// Portrait art uses the shared Storybook preview-asset catalog so component
// fixtures stay aligned with the checked-in asset manifests.
const [miraAvatar, ariaAvatar] = characterAvatarPreviewAssets;

// characterCardFixtures are the canonical reusable preview/test mocks so card
// behavior is exercised with the same Daggerheart-flavored inputs everywhere.
export const characterCardFixtures: CharacterCardFixtures = {
  full: {
    id: "char-mira",
    name: "Mira",
    portrait: {
      alt: "Portrait of Mira, a Daggerheart rogue with a guarded expression.",
      src: miraAvatar?.imageUrl,
      width: miraAvatar?.crop.widthPx,
      height: miraAvatar?.crop.heightPx,
    },
    identity: {
      kind: "PC",
      controller: "Mary",
      pronouns: "she/her",
      aliases: ["Starling"],
    },
    daggerheart: {
      summary: {
        level: 2,
        className: "Rogue",
        subclassName: "Nightwalker",
        ancestryName: "Human",
        communityName: "Slyborne",
        hp: { current: 3, max: 5 },
        stress: { current: 2, max: 6 },
        evasion: 4,
        armor: { current: 4, max: 5 },
        hope: { current: 2, max: 6 },
        feature: "Rogue's Dodge",
      },
      traits: {
        agility: "2",
        strength: "1",
        finesse: "0",
        instinct: "1",
        presence: "2",
        knowledge: "-1",
      },
    },
  },
  minimal: {
    id: "ch-a",
    name: "Aria",
    portrait: {
      alt: "Portrait of Aria, a campaign character shown in the web service roster.",
      src: ariaAvatar?.imageUrl,
      width: ariaAvatar?.crop.widthPx,
      height: ariaAvatar?.crop.heightPx,
    },
    identity: {
      kind: "PC",
      controller: "Ariadne",
    },
  },
  partial: {
    id: "ch-z",
    name: "Zara",
    portrait: {
      alt: "Portrait placeholder for Zara, a campaign character whose avatar is missing in preview fixtures.",
    },
    identity: {
      kind: "NPC",
      controller: "Moss",
      pronouns: "they/them",
    },
    daggerheart: {
      summary: {
        level: 1,
        className: "Warrior",
        ancestryName: "Human",
        hp: { current: 2, max: 4 },
      },
    },
  },
};

export type CharacterCardFixtureID = keyof typeof characterCardFixtures;
