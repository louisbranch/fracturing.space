import type { DaggerheartCharacterCardData } from "./contract";

// CharacterCardFixtures keeps the known mock-data set explicit for preview and
// test reuse across the isolated workflow.
type CharacterCardFixtures = Record<"full" | "minimal" | "partial", DaggerheartCharacterCardData>;

// The textual fixture values below are copied from current web references:
// - campaign character identity and Daggerheart summary from
//   `internal/services/web/modules/campaigns/render/detail_test.go`
// - Daggerheart full-info content from
//   `internal/services/web/modules/campaigns/render/character_creation_test.go`
//   and `internal/services/web/modules/campaigns/workflow/daggerheart/view_test.go`
// Portrait art uses stable embedded avatar assets from
// `internal/platform/assets/catalog/data/cloudinary_assets.high_fantasy.v1.json`.

const miraAvatarSheetURL =
  "https://res.cloudinary.com/fracturing-space/image/upload/v1772673703/high_fantasy/avatar_set/v1/apothecary_journeyman.png";
const ariaAvatarSheetURL =
  "https://res.cloudinary.com/fracturing-space/image/upload/v1772673710/high_fantasy/avatar_set/v1/artisan_collective_lead.png";
const portraitWidth = 512;
const portraitHeight = 768;
const portraitDeliveryWidth = 384;

// characterCardFixtures are the canonical reusable preview/test mocks so card
// behavior is exercised with the same Daggerheart-flavored inputs everywhere.
export const characterCardFixtures: CharacterCardFixtures = {
  full: {
    id: "char-mira",
    name: "Mira",
    portrait: {
      alt: "Portrait of Mira, a Daggerheart rogue with a guarded expression.",
      src: cropCloudinaryAvatarURL(miraAvatarSheetURL, {
        x: 512,
        y: 0,
        width: portraitWidth,
        height: portraitHeight,
        deliveryWidth: portraitDeliveryWidth,
      }),
      width: portraitWidth,
      height: portraitHeight,
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
      src: cropCloudinaryAvatarURL(ariaAvatarSheetURL, {
        x: 0,
        y: 768,
        width: portraitWidth,
        height: portraitHeight,
        deliveryWidth: portraitDeliveryWidth,
      }),
      width: portraitWidth,
      height: portraitHeight,
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

export function cropCloudinaryAvatarURL(
  sheetURL: string,
  input: {
    x: number;
    y: number;
    width: number;
    height: number;
    deliveryWidth: number;
  },
): string {
  const transform = [
    `c_crop,w_${input.width},h_${input.height},x_${input.x},y_${input.y}`,
    `f_auto,q_auto,dpr_auto,c_limit,w_${input.deliveryWidth}`,
  ].join("/");

  return sheetURL.replace("/image/upload/", `/image/upload/${transform}/`);
}
