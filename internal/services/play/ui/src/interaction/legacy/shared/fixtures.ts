import type { DaggerheartCharacterCardData } from "../../../systems/daggerheart/character-card/contract";
import type { DaggerheartCharacterSheetData } from "../../../systems/daggerheart/character-sheet/contract";
import type { CharacterReferenceFixtures, PlayInteractionFixtureData } from "./contract";

const defaultChat = [
  {
    messageId: "chat-1",
    actorName: "Guide",
    body: "Keep the scout talking while the rope holds.",
    sentAt: "08:13 PM",
    emphasis: "gm" as const,
  },
  {
    messageId: "chat-2",
    actorName: "Rhea",
    body: "Aria has the line. Corin, keep their eyes on you.",
    sentAt: "08:14 PM",
    emphasis: "player" as const,
  },
  {
    messageId: "chat-3",
    actorName: "Bryn",
    body: "On it. I am calling the cadence now.",
    sentAt: "08:14 PM",
    emphasis: "player" as const,
  },
];

const ariaCard: DaggerheartCharacterCardData = {
  id: "char-aria",
  name: "Aria",
  portrait: {
    alt: "Portrait placeholder for Aria, the rope runner on Storm Ledge.",
  },
  identity: {
    kind: "PC",
    controller: "Rhea",
    pronouns: "she/her",
  },
  daggerheart: {
    summary: {
      level: 2,
      className: "Guardian",
      subclassName: "Sentinel",
      ancestryName: "Human",
      communityName: "Highcliff",
      hp: { current: 3, max: 5 },
      stress: { current: 1, max: 6 },
      evasion: 10,
      armor: { current: 3, max: 4 },
      hope: { current: 2, max: 6 },
      feature: "Stand Fast",
    },
    traits: {
      agility: "1",
      strength: "2",
      finesse: "0",
      instinct: "1",
      presence: "0",
      knowledge: "1",
    },
  },
};

const corinCard: DaggerheartCharacterCardData = {
  id: "char-corin",
  name: "Corin",
  portrait: {
    alt: "Portrait placeholder for Corin, the lantern bearer on Storm Ledge.",
  },
  identity: {
    kind: "PC",
    controller: "Bryn",
    pronouns: "he/him",
  },
  daggerheart: {
    summary: {
      level: 2,
      className: "Seraph",
      subclassName: "Beacon",
      ancestryName: "Elf",
      communityName: "Harbor Ward",
      hp: { current: 4, max: 5 },
      stress: { current: 2, max: 6 },
      evasion: 11,
      armor: { current: 1, max: 3 },
      hope: { current: 4, max: 6 },
      feature: "Guiding Light",
    },
    traits: {
      agility: "0",
      strength: "0",
      finesse: "1",
      instinct: "1",
      presence: "2",
      knowledge: "1",
    },
  },
};

const ariaSheet: DaggerheartCharacterSheetData = {
  id: "char-aria",
  name: "Aria",
  portrait: {
    alt: "Portrait placeholder for Aria, a guardian braced against the storm.",
  },
  pronouns: "she/her",
  level: 2,
  className: "Guardian",
  subclassName: "Sentinel",
  ancestryName: "Human",
  communityName: "Highcliff",
  proficiency: 2,
  kind: "PC",
  controller: "Rhea",
  traits: [
    { name: "Agility", abbreviation: "AGI", value: 1, skills: ["Sprint", "Leap", "Maneuver"] },
    { name: "Strength", abbreviation: "STR", value: 2, skills: ["Lift", "Smash", "Grapple"] },
    { name: "Finesse", abbreviation: "FIN", value: 0, skills: ["Control", "Hide", "Tinker"] },
    { name: "Instinct", abbreviation: "INS", value: 1, skills: ["Perceive", "Sense", "Navigate"] },
    { name: "Presence", abbreviation: "PRE", value: 0, skills: ["Charm", "Perform", "Deceive"] },
    { name: "Knowledge", abbreviation: "KNO", value: 1, skills: ["Recall", "Analyze", "Comprehend"] },
  ],
  hp: { current: 3, max: 5 },
  stress: { current: 1, max: 6 },
  majorThreshold: 5,
  severeThreshold: 8,
  evasion: 10,
  armor: { current: 3, max: 4 },
  hope: { current: 2, max: 6 },
  hopeFeature: "Stand Fast — Spend 3 Hope to keep an ally from losing footing in the chaos.",
  classFeature: "Guardian's Intercept — Step into danger when a nearby ally would take the brunt.",
  primaryWeapon: {
    name: "Hooked Spear",
    trait: "Strength",
    range: "close",
    damageDice: "1d8",
    damageType: "physical",
    feature: "Brace",
  },
  activeArmor: {
    name: "Storm Harness",
    baseScore: 2,
    feature: "Anchored — advantage when holding position in wind or surf.",
  },
  experiences: [
    { name: "Bridge Warden", modifier: 2 },
    { name: "Storm Runner", modifier: 1 },
  ],
  domainCards: [
    { name: "Stand the Line", domain: "Valor" },
    { name: "Anchor Point", domain: "Blade" },
  ],
  gold: { handfuls: 1, bags: 0, chests: 0 },
  description: "A broad-shouldered guardian with a weather-dark cloak and a rope coil looped over one arm.",
  background: "Aria grew up protecting cliffside traders from storms and raiders, learning to hold fast where others slipped.",
  connections: "Trusts Corin's voice in a crisis and lets Bryn see the fear she hides from everyone else.",
  lifeState: "alive",
  conditions: [],
};

export const interactionCharacterFixtures: CharacterReferenceFixtures = {
  characters: [ariaCard, corinCard],
  selectedSheet: ariaSheet,
  selectedCharacterId: "char-aria",
  activeCharacterIds: ["char-aria", "char-corin"],
};

export const interactionFixtureCatalog: Record<
  | "playersOpenSingleActor"
  | "playersOpenMultiActor"
  | "gmReviewAllUnderReview"
  | "gmReviewChangesRequested"
  | "oocOpenReadyToResume"
  | "aiTurnQueued"
  | "aiTurnFailed"
  | "noActiveScene",
  PlayInteractionFixtureData
> = {
  playersOpenSingleActor: {
    title: "Single-actor player beat",
    campaignName: "Cliffside Rescue",
    sessionName: "Storm Ledge",
    systemName: "Daggerheart",
    viewerName: "Guide",
    viewerRole: "gm",
    phase: {
      status: "players",
      gmAuthorityName: "Guide",
      summary: "Corin is the only acting character in the current beat.",
    },
    scene: {
      name: "Storm Ledge",
      description: "A storm tears at the cliff path while a trapped scout clings to the far ledge.",
      gmOutputText:
        "Aria has the line, but the scout is panicking. Corin, what do you say to keep them moving?",
      gmOutputAuthor: "Guide",
      frameText:
        "Aria has the line, but the scout is panicking. Corin, what do you say to keep them moving?",
    },
    actingSet: [
      {
        id: "char-corin",
        name: "Corin",
        participantName: "Bryn",
        spotlight: true,
      },
    ],
    slots: [
      {
        participantId: "participant-bryn",
        participantName: "Bryn",
        summaryText: "Corin calls out a steady cadence and points the scout toward Aria's line.",
        characterNames: ["Corin"],
        yielded: false,
      },
    ],
    chat: defaultChat,
  },
  playersOpenMultiActor: {
    title: "Multi-actor player beat",
    campaignName: "Cliffside Rescue",
    sessionName: "Storm Ledge",
    systemName: "Daggerheart",
    viewerName: "Guide",
    viewerRole: "gm",
    phase: {
      status: "players",
      gmAuthorityName: "Guide",
      summary: "Both acting participants still own the current beat.",
    },
    scene: {
      name: "Storm Ledge",
      description: "A storm tears at the cliff path while a trapped scout clings to the far ledge.",
      gmOutputText:
        "The scout is slipping and the cliff path is crumbling under the rain. What do you do?",
      gmOutputAuthor: "Guide",
      frameText:
        "The scout is slipping and the cliff path is crumbling under the rain. What do you do?",
    },
    actingSet: [
      {
        id: "char-aria",
        name: "Aria",
        participantName: "Rhea",
        spotlight: true,
      },
      {
        id: "char-corin",
        name: "Corin",
        participantName: "Bryn",
      },
    ],
    slots: [
      {
        participantId: "participant-rhea",
        participantName: "Rhea",
        summaryText: "Aria darts for the loose mooring pin before the rope line tears free.",
        characterNames: ["Aria"],
        yielded: true,
        isViewer: false,
      },
      {
        participantId: "participant-bryn",
        participantName: "Bryn",
        summaryText: "Corin braces the line and shouts directions to the trapped scout.",
        characterNames: ["Corin"],
        yielded: false,
        isViewer: false,
      },
    ],
    chat: defaultChat,
  },
  gmReviewAllUnderReview: {
    title: "GM review",
    campaignName: "Cliffside Rescue",
    sessionName: "Storm Ledge",
    systemName: "Daggerheart",
    viewerName: "Guide",
    viewerRole: "gm",
    phase: {
      status: "gm_review",
      gmAuthorityName: "Guide",
      summary: "All acting participants have yielded and are waiting for review.",
    },
    scene: {
      name: "Storm Ledge",
      description: "A storm tears at the cliff path while a trapped scout clings to the far ledge.",
      gmOutputText:
        "The wind dies for one heartbeat, and the whole ledge feels like it is deciding whether to hold.",
      gmOutputAuthor: "Guide",
      frameText:
        "The scout is slipping and the cliff path is crumbling under the rain. What do you do?",
    },
    actingSet: [
      {
        id: "char-aria",
        name: "Aria",
        participantName: "Rhea",
      },
      {
        id: "char-corin",
        name: "Corin",
        participantName: "Bryn",
      },
    ],
    slots: [
      {
        participantId: "participant-rhea",
        participantName: "Rhea",
        summaryText: "Aria darts for the loose mooring pin before the rope line tears free.",
        characterNames: ["Aria"],
        yielded: true,
        reviewStatus: "under_review",
      },
      {
        participantId: "participant-bryn",
        participantName: "Bryn",
        summaryText: "Corin braces the line and shouts directions to the trapped scout.",
        characterNames: ["Corin"],
        yielded: true,
        reviewStatus: "under_review",
      },
    ],
    chat: defaultChat,
  },
  gmReviewChangesRequested: {
    title: "GM review with revisions",
    campaignName: "Flooded Archive",
    sessionName: "Archive Depths",
    systemName: "Daggerheart",
    viewerName: "Guide",
    viewerRole: "gm",
    phase: {
      status: "gm_review",
      gmAuthorityName: "Guide",
      summary: "One slot already has requested changes before the next reframe.",
    },
    scene: {
      name: "Flooded Archive",
      description: "Water presses at the lower stacks while the lantern light fails at the edges.",
      gmOutputText: "The floorboards sigh under the weight of the waterlogged shelves.",
      gmOutputAuthor: "Guide",
      frameText: "What do you do before the archive gives way?",
    },
    actingSet: [
      {
        id: "char-aria",
        name: "Aria",
        participantName: "Rhea",
        spotlight: true,
      },
    ],
    slots: [
      {
        participantId: "participant-rhea",
        participantName: "Rhea",
        summaryText: "Aria braces the fallen shelf against the door.",
        characterNames: ["Aria"],
        yielded: true,
        reviewStatus: "changes_requested",
        reviewReason: "Keep the lantern dry and tell me where Aria ends up.",
      },
    ],
    chat: defaultChat,
  },
  oocOpenReadyToResume: {
    title: "OOC pause with ready state",
    campaignName: "Vault Run",
    sessionName: "Sealed Vault",
    systemName: "Daggerheart",
    viewerName: "Guide",
    viewerRole: "gm",
    phase: {
      status: "gm",
      gmAuthorityName: "Guide",
      oocOpen: true,
      summary: "Scene play is paused while the table resolves a rules question.",
    },
    scene: {
      name: "Sealed Vault",
      description: "An old vault door hums with warding magic and a narrow seam of light.",
      gmOutputText: "The ward reacts to contact with the seam, not to sight or sound.",
      gmOutputAuthor: "Guide",
      frameText: "Aria, now that you know the seam is the trigger, how do you pry it open?",
    },
    actingSet: [],
    slots: [],
    ooc: {
      reason: "Clarify how the ward reacts to touch.",
      posts: [
        {
          postId: "ooc-1",
          participantName: "Rhea",
          body: "Does the ward flare if Aria uses a tool instead of bare hands?",
          emphasis: "player",
        },
        {
          postId: "ooc-2",
          participantName: "Guide",
          body: "The ward reacts to contact with the seam, not to sight or sound.",
          emphasis: "gm",
        },
        {
          postId: "ooc-3",
          participantName: "Bryn",
          body: "Then Corin can coach from a safe distance.",
          emphasis: "player",
        },
      ],
      readyParticipantNames: ["Rhea", "Bryn"],
    },
    chat: defaultChat,
  },
  aiTurnQueued: {
    title: "AI turn queued",
    campaignName: "Replay Harbor",
    sessionName: "Opening Night",
    systemName: "Daggerheart",
    viewerName: "Guide",
    viewerRole: "gm",
    phase: {
      status: "gm",
      gmAuthorityName: "Guide",
      summary: "The next authoritative GM narration is being queued for the AI seat.",
    },
    scene: {
      name: "Black Lantern",
      description: "Fog crawls under the tavern door while the dawn bell rings once outside.",
      gmOutputText:
        "The harbor master steps in, rain on his coat, and comes straight for your table.",
      gmOutputAuthor: "AI Guide",
    },
    actingSet: [],
    slots: [],
    aiTurn: {
      status: "queued",
      ownerName: "AI Guide",
      sourceLabel: "session.started bootstrap",
    },
    chat: defaultChat,
  },
  aiTurnFailed: {
    title: "AI turn failed",
    campaignName: "Replay Harbor",
    sessionName: "Opening Night",
    systemName: "Daggerheart",
    viewerName: "Guide",
    viewerRole: "gm",
    phase: {
      status: "gm",
      gmAuthorityName: "Guide",
      summary: "The authoritative AI turn failed and can be retried safely.",
    },
    scene: {
      name: "Black Lantern",
      description: "Fog crawls under the tavern door while the dawn bell rings once outside.",
      gmOutputText:
        "The harbor doors slam open before anyone can finish their drink.",
      gmOutputAuthor: "AI Guide",
    },
    actingSet: [],
    slots: [],
    aiTurn: {
      status: "failed",
      ownerName: "AI Guide",
      sourceLabel: "scene.gm_output.commit",
      lastError: "Provider timed out before the authoritative narration commit landed.",
      canRetry: true,
    },
    chat: defaultChat,
  },
  noActiveScene: {
    title: "No active scene",
    campaignName: "Replay Harbor",
    sessionName: "Opening Night",
    systemName: "Daggerheart",
    viewerName: "Guide",
    viewerRole: "gm",
    phase: {
      status: "gm",
      gmAuthorityName: "Guide",
      summary: "The session exists, but no scene has been activated yet.",
    },
    actingSet: [],
    slots: [],
    chat: defaultChat,
  },
};

export const interactionComponentFixtures = {
  phase: {
    players: interactionFixtureCatalog.playersOpenMultiActor.phase,
    gmReview: interactionFixtureCatalog.gmReviewAllUnderReview.phase,
    ooc: interactionFixtureCatalog.oocOpenReadyToResume.phase,
  },
  scene: {
    active: interactionFixtureCatalog.playersOpenMultiActor.scene!,
    empty: interactionFixtureCatalog.noActiveScene.scene,
  },
  actingSet: {
    multiActor: interactionFixtureCatalog.playersOpenMultiActor.actingSet,
    singleActor: interactionFixtureCatalog.playersOpenSingleActor.actingSet,
  },
  slots: {
    open: interactionFixtureCatalog.playersOpenMultiActor.slots,
    review: interactionFixtureCatalog.gmReviewAllUnderReview.slots,
    revisions: interactionFixtureCatalog.gmReviewChangesRequested.slots,
  },
  ooc: interactionFixtureCatalog.oocOpenReadyToResume.ooc!,
  aiTurn: {
    queued: interactionFixtureCatalog.aiTurnQueued.aiTurn!,
    failed: interactionFixtureCatalog.aiTurnFailed.aiTurn!,
  },
  chat: defaultChat,
};
