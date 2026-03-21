import { characterAvatarPreviewAssets } from "../../../storybook/preview-assets/fixtures";
import type {
  DaggerheartCharacterCardData,
  DaggerheartTrackValue,
} from "../../../systems/daggerheart/character-card/contract";
import type {
  DaggerheartCharacterSheetData,
  DaggerheartTrait,
} from "../../../systems/daggerheart/character-sheet/contract";
import type {
  PlayerHUDCharacterInspectionCatalog,
  PlayerHUDCharacterReference,
} from "./character-inspection-contract";

const [ariaAvatar, corinAvatar, sableAvatar, miraAvatar, rowanAvatar] =
  characterAvatarPreviewAssets;

function track(current: number, max: number): DaggerheartTrackValue {
  return { current, max };
}

function traits(
  agility: number,
  strength: number,
  finesse: number,
  instinct: number,
  presence: number,
  knowledge: number,
): DaggerheartTrait[] {
  return [
    { name: "Agility", abbreviation: "AGI", value: agility, skills: ["Sprint", "Leap", "Maneuver"] },
    { name: "Strength", abbreviation: "STR", value: strength, skills: ["Lift", "Smash", "Grapple"] },
    { name: "Finesse", abbreviation: "FIN", value: finesse, skills: ["Control", "Hide", "Tinker"] },
    { name: "Instinct", abbreviation: "INS", value: instinct, skills: ["Perceive", "Sense", "Navigate"] },
    { name: "Presence", abbreviation: "PRE", value: presence, skills: ["Charm", "Perform", "Deceive"] },
    { name: "Knowledge", abbreviation: "KNO", value: knowledge, skills: ["Recall", "Analyze", "Comprehend"] },
  ];
}

function createCharacter(input: {
  id: string;
  name: string;
  pronouns?: string;
  avatar: (typeof characterAvatarPreviewAssets)[number];
  className: string;
  subclassName: string;
  ancestryName: string;
  communityName: string;
  level: number;
  hp: DaggerheartTrackValue;
  stress: DaggerheartTrackValue;
  armor: DaggerheartTrackValue;
  hope: DaggerheartTrackValue;
  evasion: number;
  majorThreshold: number;
  severeThreshold: number;
  classFeature: string;
  primaryWeapon: DaggerheartCharacterSheetData["primaryWeapon"];
  secondaryWeapon: DaggerheartCharacterSheetData["secondaryWeapon"];
  activeArmor: DaggerheartCharacterSheetData["activeArmor"];
  domainCards: DaggerheartCharacterSheetData["domainCards"];
  experiences: DaggerheartCharacterSheetData["experiences"];
  gold: DaggerheartCharacterSheetData["gold"];
  description: string;
  background: string;
  connections: string;
  traitValues: [number, number, number, number, number, number];
}): {
  reference: PlayerHUDCharacterReference;
  card: DaggerheartCharacterCardData;
  sheet: DaggerheartCharacterSheetData;
} {
  const portrait = {
    alt: `Portrait of ${input.name}.`,
    src: input.avatar?.imageUrl,
    width: input.avatar?.crop.widthPx,
    height: input.avatar?.crop.heightPx,
  };

  return {
    reference: {
      id: input.id,
      name: input.name,
      avatarUrl: input.avatar?.imageUrl,
    },
    card: {
      id: input.id,
      name: input.name,
      portrait,
      identity: {
        kind: "PC",
        pronouns: input.pronouns,
      },
      daggerheart: {
        summary: {
          level: input.level,
          className: input.className,
          subclassName: input.subclassName,
          ancestryName: input.ancestryName,
          communityName: input.communityName,
          hp: input.hp,
          stress: input.stress,
          evasion: input.evasion,
          armor: input.armor,
          hope: input.hope,
          feature: input.classFeature,
        },
        traits: {
          agility: String(input.traitValues[0]),
          strength: String(input.traitValues[1]),
          finesse: String(input.traitValues[2]),
          instinct: String(input.traitValues[3]),
          presence: String(input.traitValues[4]),
          knowledge: String(input.traitValues[5]),
        },
      },
    },
    sheet: {
      id: input.id,
      name: input.name,
      portrait,
      pronouns: input.pronouns,
      level: input.level,
      className: input.className,
      subclassName: input.subclassName,
      ancestryName: input.ancestryName,
      communityName: input.communityName,
      proficiency: 2,
      traits: traits(...input.traitValues),
      hp: input.hp,
      stress: input.stress,
      majorThreshold: input.majorThreshold,
      severeThreshold: input.severeThreshold,
      evasion: input.evasion,
      armor: input.armor,
      hope: input.hope,
      hopeFeature: `${input.classFeature} Spend Hope to press the advantage at a crucial moment.`,
      classFeature: input.classFeature,
      primaryWeapon: input.primaryWeapon,
      secondaryWeapon: input.secondaryWeapon,
      activeArmor: input.activeArmor,
      experiences: input.experiences,
      domainCards: input.domainCards,
      gold: input.gold,
      description: input.description,
      background: input.background,
      connections: input.connections,
      lifeState: "alive",
      conditions: [],
      kind: "PC",
    },
  };
}

const aria = createCharacter({
  id: "char-aria",
  name: "Aria",
  pronouns: "she/her",
  avatar: ariaAvatar,
  className: "Guardian",
  subclassName: "Vanguard",
  ancestryName: "Human",
  communityName: "Highborne",
  level: 3,
  hp: track(4, 6),
  stress: track(2, 6),
  armor: track(5, 6),
  hope: track(3, 6),
  evasion: 11,
  majorThreshold: 6,
  severeThreshold: 9,
  classFeature: "Bulwark Stance",
  primaryWeapon: {
    name: "Pryblade Spear",
    trait: "Strength",
    range: "melee",
    damageDice: "1d10",
    damageType: "physical",
    feature: "Brace",
  },
  secondaryWeapon: {
    name: "Holdout Knife",
    trait: "Finesse",
    range: "very close",
    damageDice: "1d6",
    damageType: "physical",
  },
  activeArmor: {
    name: "Plate Coat",
    baseScore: 3,
    feature: "Reinforced at the shoulders.",
  },
  domainCards: [
    { name: "Stand Firm", domain: "Valor" },
    { name: "Hold the Line", domain: "Blade" },
    { name: "Turn the Blow", domain: "Bone" },
  ],
  experiences: [
    { name: "Breach Specialist", modifier: 2 },
    { name: "Siege Veteran", modifier: 1 },
    { name: "Reads old wards", modifier: 1 },
  ],
  gold: { handfuls: 2, bags: 1, chests: 0 },
  description:
    "Aria carries herself like a shield wall distilled into one person: broad-shouldered, watchful, and hard to move once planted.",
  background:
    "She spent years opening sealed keeps for hire, learning how to respect the weight of old wards without letting them dictate the pace.",
  connections:
    "Corin trusts her timing. Sable trusts her to hold the line when plans fail.",
  traitValues: [0, 2, 1, 1, 0, 1],
});

const corin = createCharacter({
  id: "char-corin",
  name: "Corin",
  pronouns: "he/him",
  avatar: corinAvatar,
  className: "Sage",
  subclassName: "Runekeeper",
  ancestryName: "Elf",
  communityName: "Lorebound",
  level: 3,
  hp: track(3, 5),
  stress: track(3, 6),
  armor: track(2, 4),
  hope: track(4, 6),
  evasion: 10,
  majorThreshold: 5,
  severeThreshold: 8,
  classFeature: "Pattern Recall",
  primaryWeapon: {
    name: "Etched Staff",
    trait: "Knowledge",
    range: "melee",
    damageDice: "1d8",
    damageType: "arcane",
    feature: "Channel",
  },
  secondaryWeapon: {
    name: "Warding Chalk",
    trait: "Instinct",
    range: "close",
    damageDice: "1d6",
    damageType: "arcane",
  },
  activeArmor: {
    name: "Scholar's Coat",
    baseScore: 1,
    feature: "Warded lining.",
  },
  domainCards: [
    { name: "Sigil Sight", domain: "Codex" },
    { name: "Echo Counter", domain: "Midnight" },
    { name: "Widen the Pattern", domain: "Arcana" },
  ],
  experiences: [
    { name: "Runic Archive", modifier: 2 },
    { name: "Lantern discipline", modifier: 1 },
    { name: "Temple surveyor" },
  ],
  gold: { handfuls: 1, bags: 1, chests: 0 },
  description:
    "Corin is lean and restless, with ink-dark curls and fingers that are always measuring invisible distances in the air.",
  background:
    "A former apprentice archivist, he learned to survive by understanding where a pattern will fail before anyone else notices it.",
  connections:
    "Aria follows his counts without argument. Rowan calls him a worrywart and listens anyway.",
  traitValues: [0, -1, 1, 2, 0, 2],
});

const sable = createCharacter({
  id: "char-sable",
  name: "Sable",
  pronouns: "they/them",
  avatar: sableAvatar,
  className: "Rogue",
  subclassName: "Nightwalker",
  ancestryName: "Human",
  communityName: "Slyborne",
  level: 2,
  hp: track(3, 5),
  stress: track(2, 6),
  armor: track(2, 4),
  hope: track(2, 6),
  evasion: 12,
  majorThreshold: 5,
  severeThreshold: 8,
  classFeature: "Shadow Drift",
  primaryWeapon: {
    name: "Gallery Dagger",
    trait: "Finesse",
    range: "very close",
    damageDice: "1d8",
    damageType: "physical",
    feature: "Silent",
  },
  secondaryWeapon: {
    name: "Throwing Blade",
    trait: "Agility",
    range: "close",
    damageDice: "1d6",
    damageType: "physical",
  },
  activeArmor: {
    name: "Quiet Leathers",
    baseScore: 1,
    feature: "Soft-soled and dark dyed.",
  },
  domainCards: [
    { name: "Cut the Lantern", domain: "Midnight" },
    { name: "Vanish in Motion", domain: "Grace" },
    { name: "Ghost Step", domain: "Shadow" },
  ],
  experiences: [
    { name: "Upper gallery scout", modifier: 2 },
    { name: "Knife work", modifier: 1 },
    { name: "Keeps low" },
  ],
  gold: { handfuls: 2, bags: 0, chests: 0 },
  description:
    "Sable moves like they are apologizing to the floorboards for having to touch them at all.",
  background:
    "They grew up in balcony rafters, service passages, and the blind corners that good houses forget to guard.",
  connections:
    "Mira shares contacts with them. Aria trusts them to cover the retreat.",
  traitValues: [2, 0, 2, 1, 1, 0],
});

const mira = createCharacter({
  id: "char-mira",
  name: "Mira",
  pronouns: "she/her",
  avatar: miraAvatar,
  className: "Rogue",
  subclassName: "Nightwalker",
  ancestryName: "Human",
  communityName: "Slyborne",
  level: 2,
  hp: track(3, 5),
  stress: track(2, 6),
  armor: track(4, 5),
  hope: track(2, 6),
  evasion: 10,
  majorThreshold: 5,
  severeThreshold: 8,
  classFeature: "Sneak Attack",
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
    feature: "Light and travel-worn.",
  },
  domainCards: [
    { name: "Vanishing Dodge", domain: "Midnight" },
    { name: "Cloaking Blast", domain: "Arcana" },
    { name: "Bolt Beacon", domain: "Splendor" },
  ],
  experiences: [
    { name: "Wanderer", modifier: 2 },
    { name: "Streetwise" },
    { name: "Scholar", modifier: -1 },
  ],
  gold: { handfuls: 3, bags: 1, chests: 0 },
  description:
    "Mira has sharp dark eyes, cropped black hair, and the alert posture of someone who hates standing where others can corner her.",
  background:
    "She learned early that silence and speed were worth more than coin, and she has kept both close ever since.",
  connections:
    "Sable still owes her from the bridge incident. Rowan keeps trying to teach her patience.",
  traitValues: [2, 1, 0, 1, 2, -1],
});

const rowan = createCharacter({
  id: "char-rowan",
  name: "Rowan",
  pronouns: "he/they",
  avatar: rowanAvatar,
  className: "Guardian",
  subclassName: "Wildkeeper",
  ancestryName: "Firbolg",
  communityName: "Hearthwild",
  level: 2,
  hp: track(5, 6),
  stress: track(1, 6),
  armor: track(3, 5),
  hope: track(3, 6),
  evasion: 9,
  majorThreshold: 6,
  severeThreshold: 9,
  classFeature: "Steady Heart",
  primaryWeapon: {
    name: "Oak Maul",
    trait: "Strength",
    range: "melee",
    damageDice: "1d10",
    damageType: "physical",
    feature: "Heavy",
  },
  secondaryWeapon: {
    name: "Travel Sling",
    trait: "Instinct",
    range: "far",
    damageDice: "1d6",
    damageType: "physical",
  },
  activeArmor: {
    name: "Layered Hide",
    baseScore: 2,
    feature: "Weathered and reliable.",
  },
  domainCards: [
    { name: "Root Yourself", domain: "Stone" },
    { name: "Answer the Charge", domain: "Valor" },
    { name: "Carry Them Out", domain: "Bone" },
  ],
  experiences: [
    { name: "Pack mule humor", modifier: 1 },
    { name: "Courtyard brawler", modifier: 1 },
    { name: "Night watch" },
  ],
  gold: { handfuls: 1, bags: 0, chests: 0 },
  description:
    "Rowan looks carved from warm timber, broad-handed and calm until the moment action becomes necessary.",
  background:
    "They left the deep roads to see whether the world beyond their hearth was really as fragile as travelers made it sound.",
  connections:
    "Aria counts on them for extraction. Corin trusts Rowan to stay put when the rest of the room starts moving.",
  traitValues: [0, 2, 0, 2, 1, 0],
});

export const playerHUDCharacterCatalog = {
  aria: aria.reference,
  corin: corin.reference,
  sable: sable.reference,
  mira: mira.reference,
  rowan: rowan.reference,
} satisfies Record<string, PlayerHUDCharacterReference>;

export const playerHUDCharacterInspectionCatalog: PlayerHUDCharacterInspectionCatalog = {
  [aria.reference.id]: { system: "daggerheart", card: aria.card, sheet: aria.sheet },
  [corin.reference.id]: { system: "daggerheart", card: corin.card, sheet: corin.sheet },
  [sable.reference.id]: { system: "daggerheart", card: sable.card, sheet: sable.sheet },
  [mira.reference.id]: { system: "daggerheart", card: mira.card, sheet: mira.sheet },
  [rowan.reference.id]: { system: "daggerheart", card: rowan.card, sheet: rowan.sheet },
};
