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
    { id: "domain_card.valor-stand-firm", name: "Stand Firm", domain: "Valor", featureText: "Plant yourself and refuse to be moved until the exchange breaks." },
    { id: "domain_card.blade-hold-the-line", name: "Hold the Line", domain: "Blade", featureText: "Meet the charge head-on and make any advance through your reach costly." },
    { id: "domain_card.bone-turn-the-blow", name: "Turn the Blow", domain: "Bone", featureText: "Angle the hit away and bleed momentum off the strike before it reaches an ally." },
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
    { id: "domain_card.codex-sigil-sight", name: "Sigil Sight", domain: "Codex", featureText: "Read the active glyphwork at a glance and expose the weak point in the sequence." },
    { id: "domain_card.midnight-echo-counter", name: "Echo Counter", domain: "Midnight", featureText: "Answer hostile magic with a reflected afterimage that disrupts its timing." },
    { id: "domain_card.arcana-widen-the-pattern", name: "Widen the Pattern", domain: "Arcana", featureText: "Expand the spell lattice so nearby allies can move through the same opening." },
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
    { id: "domain_card.midnight-cut-the-lantern", name: "Cut the Lantern", domain: "Midnight", featureText: "Snuff the nearest light source and turn the scene into your terrain." },
    { id: "domain_card.grace-vanish-in-motion", name: "Vanish in Motion", domain: "Grace", featureText: "Keep moving through the chaos until no one can agree where you went." },
    { id: "domain_card.shadow-ghost-step", name: "Ghost Step", domain: "Shadow", featureText: "Cross a dangerous gap in a blur and reappear where the guard line is weakest." },
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
    { id: "domain_card.midnight-vanishing-dodge", name: "Vanishing Dodge", domain: "Midnight", featureText: "Slip out of reach and reset your footing before the enemy can follow through." },
    { id: "domain_card.arcana-cloaking-blast", name: "Cloaking Blast", domain: "Arcana", featureText: "Throw a burst of arcane cover into the lane and disappear behind it." },
    { id: "domain_card.splendor-bolt-beacon", name: "Bolt Beacon", domain: "Splendor", featureText: "Tag the opening with radiant force so everyone else can press the same weakness." },
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
    { id: "domain_card.stone-root-yourself", name: "Root Yourself", domain: "Stone", featureText: "Drop your weight and become the anchor point the rest of the retreat can pivot around." },
    { id: "domain_card.valor-answer-the-charge", name: "Answer the Charge", domain: "Valor", featureText: "Step into the rush and make the aggressor deal with you first." },
    { id: "domain_card.bone-carry-them-out", name: "Carry Them Out", domain: "Bone", featureText: "Lift a fallen ally clear of the melee without surrendering the line." },
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
