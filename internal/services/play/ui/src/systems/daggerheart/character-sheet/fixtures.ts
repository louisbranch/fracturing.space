import { characterAvatarPreviewAssets } from "../../../storybook/preview-assets/fixtures";
import type { DaggerheartCharacterSheetData } from "./contract";

// CharacterSheetFixtures keeps the known mock-data set explicit for preview and
// test reuse across the isolated workflow.
type CharacterSheetFixtures = Record<"full" | "damaged", DaggerheartCharacterSheetData>;

const [miraAvatar] = characterAvatarPreviewAssets;

// characterSheetFixtures are the canonical reusable preview/test mocks so the
// sheet is exercised with rich Daggerheart-flavored inputs everywhere.
export const characterSheetFixtures: CharacterSheetFixtures = {
  full: {
    id: "char-mira",
    name: "Mira",
    portrait: {
      alt: "Portrait of Mira, a Daggerheart rogue with a guarded expression.",
      src: miraAvatar?.imageUrl,
      width: miraAvatar?.crop.widthPx,
      height: miraAvatar?.crop.heightPx,
    },
    pronouns: "she/her",
    level: 2,
    className: "Rogue",
    subclassName: "Nightwalker",
    ancestryName: "Human",
    communityName: "Slyborne",
    proficiency: 2,
    kind: "PC",
    controller: "Mary",

    traits: [
      { name: "Agility", abbreviation: "AGI", value: 2, skills: ["Sprint", "Leap", "Maneuver"] },
      { name: "Strength", abbreviation: "STR", value: 1, skills: ["Lift", "Smash", "Grapple"] },
      { name: "Finesse", abbreviation: "FIN", value: 0, skills: ["Control", "Hide", "Tinker"] },
      { name: "Instinct", abbreviation: "INS", value: 1, skills: ["Perceive", "Sense", "Navigate"] },
      { name: "Presence", abbreviation: "PRE", value: 2, skills: ["Charm", "Perform", "Deceive"] },
      { name: "Knowledge", abbreviation: "KNO", value: -1, skills: ["Recall", "Analyze", "Comprehend"] },
    ],

    hp: { current: 3, max: 5 },
    stress: { current: 2, max: 6 },
    majorThreshold: 5,
    severeThreshold: 8,

    evasion: 10,
    armor: { current: 4, max: 5 },

    hope: { current: 2, max: 6 },
    hopeFeature: "Rogue's Dodge — Spend 3 Hope to gain +2 Evasion until an attack succeeds against you; otherwise it lasts until your next rest.",

    classFeature: "Sneak Attack — When you have advantage on a melee attack, deal an extra 1d8 damage.",

    primaryWeapon: {
      name: "Sword",
      trait: "Finesse",
      range: "melee",
      damageDice: "1d8",
      damageType: "physical",
      feature: "Versatile",
    },
    secondaryWeapon: {
      name: "Dagger",
      trait: "Finesse",
      range: "very close",
      damageDice: "1d6",
      damageType: "physical",
    },
    activeArmor: {
      name: "Leather",
      baseScore: 2,
      feature: "Light — does not impose disadvantage on stealth.",
    },

    experiences: [
      { name: "Wanderer", modifier: 2 },
      { name: "Streetwise" },
      { name: "Scholar", modifier: -1 },
    ],
    domainCards: [
      { name: "Vanishing Dodge", domain: "Midnight" },
      { name: "Cloaking Blast", domain: "Arcana" },
      { name: "Bolt Beacon", domain: "Splendor" },
    ],

    gold: { handfuls: 3, bags: 1, chests: 0 },

    description: "A wiry young woman with sharp dark eyes, cropped black hair, and a thin scar running from her left ear to her jaw. She wears a battered leather coat over close-fitting travelling clothes and moves with a cat-like economy of motion.",

    background: "Mira grew up in the back alleys of Havenport, learning early that silence and speed were worth more than any coin. She ran with the Starling crew until a job went wrong and she found herself alone, wanted, and headed for the frontier.",
    connections: "Owes a debt to Aldric the fence. Has a complicated history with the Slyborne Underground. Trusted by the party after the bridge incident.",

    lifeState: "alive",
    conditions: [],
  },

  damaged: {
    id: "char-dmg",
    name: "Mira",
    portrait: {
      alt: "Portrait of Mira, a Daggerheart rogue with a guarded expression.",
      src: miraAvatar?.imageUrl,
      width: miraAvatar?.crop.widthPx,
      height: miraAvatar?.crop.heightPx,
    },
    pronouns: "she/her",
    level: 2,
    className: "Rogue",
    subclassName: "Nightwalker",
    ancestryName: "Human",
    communityName: "Slyborne",

    traits: [
      { name: "Agility", abbreviation: "AGI", value: 2, skills: ["Sprint", "Leap", "Maneuver"] },
      { name: "Strength", abbreviation: "STR", value: 1, skills: ["Lift", "Smash", "Grapple"] },
      { name: "Finesse", abbreviation: "FIN", value: 0, skills: ["Control", "Hide", "Tinker"] },
      { name: "Instinct", abbreviation: "INS", value: 1, skills: ["Perceive", "Sense", "Navigate"] },
      { name: "Presence", abbreviation: "PRE", value: 2, skills: ["Charm", "Perform", "Deceive"] },
      { name: "Knowledge", abbreviation: "KNO", value: -1, skills: ["Recall", "Analyze", "Comprehend"] },
    ],

    hp: { current: 1, max: 5 },
    stress: { current: 5, max: 6 },
    majorThreshold: 5,
    severeThreshold: 8,

    evasion: 10,
    armor: { current: 0, max: 5 },

    hope: { current: 0, max: 6 },
    hopeFeature: "Rogue's Dodge — Spend 3 Hope to gain +2 Evasion until an attack succeeds against you; otherwise it lasts until your next rest.",

    lifeState: "unconscious",
    conditions: ["Frightened", "Vulnerable"],
  },
};

export type CharacterSheetFixtureID = keyof typeof characterSheetFixtures;
