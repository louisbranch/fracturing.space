import { participantAvatarPreviewAssets } from "../../../../storybook/preview-assets/fixtures";
import {
  playerHUDCharacterCatalog,
  playerHUDCharacterInspectionCatalog,
} from "../../shared/character-inspection-fixtures";
import type {
  OnStageCharacterSummary,
  OnStageGMInteraction,
  OnStageGMInteractionIllustration,
  OnStageParticipant,
  OnStageScene,
  OnStageSlot,
  OnStageState,
} from "./contract";

const [rheaAvatar, brynAvatar, guideAvatar] = participantAvatarPreviewAssets;

export const onStageCharacterCatalog = {
  aria: playerHUDCharacterCatalog.aria,
  corin: playerHUDCharacterCatalog.corin,
  sable: playerHUDCharacterCatalog.sable,
  mira: playerHUDCharacterCatalog.mira,
  rowan: playerHUDCharacterCatalog.rowan,
} satisfies Record<string, OnStageCharacterSummary>;

function sceneCharacters(...ids: (keyof typeof onStageCharacterCatalog)[]): OnStageCharacterSummary[] {
  return ids.map((id) => onStageCharacterCatalog[id]);
}

export const onStageParticipants: OnStageParticipant[] = [
  {
    id: "p-rhea",
    name: "Rhea",
    role: "player",
    avatarUrl: rheaAvatar?.imageUrl,
    characters: [onStageCharacterCatalog.aria],
    railStatus: "active",
  },
  {
    id: "p-bryn",
    name: "Bryn",
    role: "player",
    avatarUrl: brynAvatar?.imageUrl,
    characters: [onStageCharacterCatalog.corin],
    railStatus: "waiting",
  },
  {
    id: "p-guide",
    name: "Guide",
    role: "gm",
    avatarUrl: guideAvatar?.imageUrl,
    characters: [],
    railStatus: "waiting",
    ownsGMAuthority: true,
  },
];

function slot(input: Omit<OnStageSlot, "id"> & { id?: string }): OnStageSlot {
  return {
    id: input.id ?? `${input.participantId}-${input.characters.map((character) => character.id).join("-")}`,
    ...input,
  };
}

function interaction(input: OnStageGMInteraction): OnStageGMInteraction {
  return input;
}

function scene(input: OnStageScene): OnStageScene {
  return input;
}

// Restored from the On Stage vNext preview so the stable card keeps a wide
// illustration example even though this asset is preview-only fixture data.
export const lanternVignetteIllustration: OnStageGMInteractionIllustration = {
  imageUrl:
    "https://res.cloudinary.com/fracturing-space/image/upload/v1773860418/high_fantasy/event_vignette/v1/lantern_in_the_dark.png",
  alt: "A storm lantern burning in darkness while rigging and rain close around it.",
  sizeHint: "wide",
};

export const archerGuardIllustration: OnStageGMInteractionIllustration = {
  imageUrl:
    "https://res.cloudinary.com/fracturing-space/image/upload/v1773619370/high_fantasy/daggerheart_adversary_illustration/v1/archer_guard.png",
  alt: "An archer guard drawing and aiming from a fortified position.",
  caption: "Enemy attack illustration example.",
  sizeHint: "compact",
};

const sealedVaultScene = scene({
  id: "scene-sealed-vault",
  name: "Sealed Vault",
  description:
    "A humming ward seals the old vault while a hairline seam catches the lantern light.",
  characters: sceneCharacters("aria", "corin"),
  resolvedInteractionCount: 1,
});

const sealedVaultCurrentInteraction = interaction({
  id: "gmint-sealed-vault-current",
  title: "At the Vault Seam",
  characterIds: ["aria", "corin"],
  illustration: lanternVignetteIllustration,
  beats: [
    {
      id: "beat-sealed-vault-fiction",
      type: "fiction",
      text:
        "The vault door hums with old warding magic, and the seam flashes whenever a hand drifts too close. The light inside the crack is too steady to be fire and too warm to be moonlight, and every pulse travels the bronze frame like a warning looking for a louder shape.",
    },
    {
      id: "beat-sealed-vault-prompt",
      type: "prompt",
      text:
        "The ward crackles when either of you nears the seam. What do you do before the alarm wakes the keep?",
    },
  ],
});

const sealedVaultHistory = [
  interaction({
    id: "gmint-sealed-vault-history-1",
    title: "The Warning Lattice",
    characterIds: ["aria", "corin"],
    beats: [
      {
        id: "beat-warning-lattice-guidance",
        type: "guidance",
        text:
          "The lower glyph ring is already unstable, so force will only make the ward louder. If the seam opens at all, it needs to happen on the lattice's rhythm rather than against it.",
      },
    ],
  }),
];

const tightenedVaultInteraction = interaction({
  id: "gmint-sealed-vault-tightened",
  title: "Tighten the Approach",
  characterIds: ["aria"],
  beats: [
    {
      id: "beat-tighten-consequence",
      type: "consequence",
      text:
        "The seam parts for a breath, then snaps hard toward the tool instead of away from it. The ward has not fully escalated yet, but it has started reading Aria's leverage as a threat instead of a probe.",
    },
    {
      id: "beat-tighten-prompt",
      type: "prompt",
      text:
        "Aria needs a tighter commitment now. How does she keep contact off the seam itself while still taking advantage of the warped bronze lip?",
    },
  ],
});

const moonlitCourtyardScene = scene({
  id: "scene-moonlit-courtyard",
  name: "Moonlit Courtyard",
  description:
    "A fountain masks quiet footfalls while four shadows move between the arches.",
  characters: sceneCharacters("aria", "sable", "mira", "rowan"),
  resolvedInteractionCount: 1,
});

const moonlitCourtyardInteraction = interaction({
  id: "gmint-moonlit-courtyard-current",
  title: "Across the Courtyard",
  characterIds: ["aria", "sable", "mira", "rowan"],
  beats: [
    {
      id: "beat-courtyard-fiction",
      type: "fiction",
      text:
        "The courtyard is quiet for now, but the keep windows are still lit above the fountain. Lanternlight breaks across the wet flagstones in narrow bands, and every archway gives you cover only until the patrol loops past it again.",
    },
    {
      id: "beat-courtyard-prompt",
      type: "prompt",
      text:
        "Your whole crew is in position. How do they move through the courtyard before the patrol turns back?",
    },
  ],
});

const defaultMechanicsExtension = {
  label: "System actions",
  description: "System-specific mechanics can appear here when the current beat needs them.",
} as const;

const multiCharacterMechanicsExtension = {
  label: "System actions",
  description: "System-specific mechanics can appear here without changing the participant-owned slot model.",
} as const;

export const onStageFixtureCatalog: Record<
  | "waitingOnGM"
  | "actingEmpty"
  | "viewerPosted"
  | "yieldedWaiting"
  | "changesRequested"
  | "oocBlocked"
  | "aiThinking"
  | "aiFailed"
  | "multiCharacterOwner",
  OnStageState
> = {
  waitingOnGM: {
    mode: "waiting-on-gm",
    aiStatus: "idle",
    scene: sealedVaultScene,
    currentInteraction: sealedVaultCurrentInteraction,
    interactionHistory: sealedVaultHistory,
    viewerParticipantId: "p-rhea",
    actingParticipantIds: [],
    gmAuthorityParticipantId: "p-guide",
    participants: onStageParticipants.map((participant) => ({
      ...participant,
      railStatus: "waiting",
    })),
    slots: [],
    characterInspectionCatalog: playerHUDCharacterInspectionCatalog,
    viewerControls: {
      canSubmit: false,
      canSubmitAndYield: false,
      canYield: false,
      canUnyield: false,
      disabledReason: "Waiting for the GM to open the next beat.",
    },
    mechanicsExtension: defaultMechanicsExtension,
  },
  actingEmpty: {
    mode: "acting",
    aiStatus: "idle",
    scene: { ...sealedVaultScene, resolvedInteractionCount: 0 },
    currentInteraction: sealedVaultCurrentInteraction,
    interactionHistory: [],
    viewerParticipantId: "p-rhea",
    actingParticipantIds: ["p-rhea", "p-bryn"],
    gmAuthorityParticipantId: "p-guide",
    participants: onStageParticipants.map((participant) =>
      participant.id === "p-rhea" || participant.id === "p-bryn"
        ? { ...participant, railStatus: "active" }
        : participant,
    ),
    slots: [
      slot({
        participantId: "p-bryn",
        characters: [onStageCharacterCatalog.corin],
        body: "Corin stays back from the seam and studies the runes for a safe approach.",
        updatedAt: "2026-03-19T20:32:00Z",
        yielded: false,
        reviewState: "open",
      }),
    ],
    characterInspectionCatalog: playerHUDCharacterInspectionCatalog,
    viewerControls: {
      canSubmit: true,
      canSubmitAndYield: true,
      canYield: true,
      canUnyield: false,
    },
    mechanicsExtension: defaultMechanicsExtension,
  },
  viewerPosted: {
    mode: "acting",
    aiStatus: "idle",
    scene: sealedVaultScene,
    currentInteraction: sealedVaultCurrentInteraction,
    interactionHistory: sealedVaultHistory,
    viewerParticipantId: "p-rhea",
    actingParticipantIds: ["p-rhea", "p-bryn"],
    gmAuthorityParticipantId: "p-guide",
    participants: onStageParticipants.map((participant) =>
      participant.id === "p-rhea" || participant.id === "p-bryn"
        ? { ...participant, railStatus: "active" }
        : participant,
    ),
    slots: [
      slot({
        participantId: "p-rhea",
        characters: [onStageCharacterCatalog.aria],
        body: "Aria hooks a pry tool into the seam and braces for the ward's recoil.",
        updatedAt: "2026-03-19T20:31:00Z",
        yielded: false,
        reviewState: "open",
      }),
      slot({
        participantId: "p-bryn",
        characters: [onStageCharacterCatalog.corin],
        body: "Corin shields the lantern and points out a break in the glyph pattern.",
        updatedAt: "2026-03-19T20:32:00Z",
        yielded: false,
        reviewState: "open",
      }),
    ],
    characterInspectionCatalog: playerHUDCharacterInspectionCatalog,
    viewerControls: {
      canSubmit: true,
      canSubmitAndYield: true,
      canYield: true,
      canUnyield: false,
    },
    mechanicsExtension: defaultMechanicsExtension,
  },
  yieldedWaiting: {
    mode: "yielded-waiting",
    aiStatus: "idle",
    scene: sealedVaultScene,
    currentInteraction: sealedVaultCurrentInteraction,
    interactionHistory: sealedVaultHistory,
    viewerParticipantId: "p-rhea",
    actingParticipantIds: ["p-rhea", "p-bryn"],
    gmAuthorityParticipantId: "p-guide",
    participants: [
      { ...onStageParticipants[0], railStatus: "yielded" },
      { ...onStageParticipants[1], railStatus: "active" },
      onStageParticipants[2],
    ],
    slots: [
      slot({
        participantId: "p-rhea",
        characters: [onStageCharacterCatalog.aria],
        body: "Aria hooks a pry tool into the seam and braces for the ward's recoil.",
        updatedAt: "2026-03-19T20:31:00Z",
        yielded: true,
        reviewState: "under-review",
      }),
      slot({
        participantId: "p-bryn",
        characters: [onStageCharacterCatalog.corin],
        body: "Corin watches the glyph pattern and calls out where the ward thins.",
        updatedAt: "2026-03-19T20:32:00Z",
        yielded: false,
        reviewState: "open",
      }),
    ],
    characterInspectionCatalog: playerHUDCharacterInspectionCatalog,
    viewerControls: {
      canSubmit: false,
      canSubmitAndYield: false,
      canYield: false,
      canUnyield: true,
      disabledReason: "You have already yielded. Unyield if you need to revise before the beat closes.",
    },
    mechanicsExtension: defaultMechanicsExtension,
  },
  changesRequested: {
    mode: "changes-requested",
    aiStatus: "idle",
    scene: sealedVaultScene,
    currentInteraction: tightenedVaultInteraction,
    interactionHistory: sealedVaultHistory,
    viewerParticipantId: "p-rhea",
    actingParticipantIds: ["p-rhea"],
    gmAuthorityParticipantId: "p-guide",
    participants: [
      { ...onStageParticipants[0], railStatus: "changes-requested" },
      { ...onStageParticipants[1], railStatus: "waiting" },
      onStageParticipants[2],
    ],
    slots: [
      slot({
        participantId: "p-rhea",
        characters: [onStageCharacterCatalog.aria],
        body: "Aria hooks a pry tool into the seam and braces for the ward's recoil.",
        updatedAt: "2026-03-19T20:31:00Z",
        yielded: false,
        reviewState: "changes-requested",
        reviewReason: "Commit to how Aria keeps contact off the seam itself.",
      }),
    ],
    characterInspectionCatalog: playerHUDCharacterInspectionCatalog,
    viewerControls: {
      canSubmit: true,
      canSubmitAndYield: true,
      canYield: false,
      canUnyield: false,
    },
    mechanicsExtension: defaultMechanicsExtension,
  },
  oocBlocked: {
    mode: "ooc-blocked",
    aiStatus: "idle",
    scene: sealedVaultScene,
    currentInteraction: sealedVaultCurrentInteraction,
    interactionHistory: sealedVaultHistory,
    oocReason: "Clarify whether tools touching the seam trigger the ward.",
    viewerParticipantId: "p-rhea",
    actingParticipantIds: ["p-rhea", "p-bryn"],
    gmAuthorityParticipantId: "p-guide",
    participants: onStageParticipants.map((participant) => ({
      ...participant,
      railStatus: "waiting",
    })),
    slots: [
      slot({
        participantId: "p-rhea",
        characters: [onStageCharacterCatalog.aria],
        body: "Aria starts to line up the tool, then pauses for clarification.",
        updatedAt: "2026-03-19T20:31:00Z",
        yielded: false,
        reviewState: "open",
      }),
    ],
    characterInspectionCatalog: playerHUDCharacterInspectionCatalog,
    viewerControls: {
      canSubmit: false,
      canSubmitAndYield: false,
      canYield: false,
      canUnyield: false,
      disabledReason: "Backstage OOC is open. Resolve the ruling there before acting on-stage.",
    },
    mechanicsExtension: defaultMechanicsExtension,
  },
  aiThinking: {
    mode: "waiting-on-gm",
    aiStatus: "running",
    scene: sealedVaultScene,
    currentInteraction: sealedVaultCurrentInteraction,
    interactionHistory: sealedVaultHistory,
    viewerParticipantId: "p-rhea",
    actingParticipantIds: [],
    gmAuthorityParticipantId: "p-guide",
    participants: onStageParticipants.map((participant) => ({
      ...participant,
      railStatus: "waiting",
    })),
    slots: [],
    characterInspectionCatalog: playerHUDCharacterInspectionCatalog,
    viewerControls: {
      canSubmit: false,
      canSubmitAndYield: false,
      canYield: false,
      canUnyield: false,
      disabledReason: "The AI GM is preparing the next beat.",
    },
    mechanicsExtension: defaultMechanicsExtension,
  },
  aiFailed: {
    mode: "waiting-on-gm",
    aiStatus: "failed",
    scene: sealedVaultScene,
    currentInteraction: sealedVaultCurrentInteraction,
    interactionHistory: sealedVaultHistory,
    viewerParticipantId: "p-rhea",
    actingParticipantIds: [],
    gmAuthorityParticipantId: "p-guide",
    participants: onStageParticipants.map((participant) => ({
      ...participant,
      railStatus: "waiting",
    })),
    slots: [],
    characterInspectionCatalog: playerHUDCharacterInspectionCatalog,
    viewerControls: {
      canSubmit: false,
      canSubmitAndYield: false,
      canYield: false,
      canUnyield: false,
      disabledReason: "The next beat is delayed while GM authority reorients.",
    },
    mechanicsExtension: defaultMechanicsExtension,
  },
  multiCharacterOwner: {
    mode: "acting",
    aiStatus: "idle",
    scene: moonlitCourtyardScene,
    currentInteraction: moonlitCourtyardInteraction,
    interactionHistory: [],
    viewerParticipantId: "p-rhea",
    actingParticipantIds: ["p-rhea"],
    gmAuthorityParticipantId: "p-guide",
    participants: [
      {
        ...onStageParticipants[0],
        characters: sceneCharacters("aria", "sable", "mira", "rowan"),
        railStatus: "active",
      },
      { ...onStageParticipants[2], railStatus: "waiting" },
    ],
    slots: [
      slot({
        participantId: "p-rhea",
        characters: sceneCharacters("aria", "sable", "mira", "rowan"),
        body: "Aria watches the archway while Sable crosses to the fountain and Mira guides Rowan through the blind side of the lantern light.",
        updatedAt: "2026-03-19T20:48:00Z",
        yielded: false,
        reviewState: "open",
      }),
    ],
    characterInspectionCatalog: playerHUDCharacterInspectionCatalog,
    viewerControls: {
      canSubmit: true,
      canSubmitAndYield: true,
      canYield: true,
      canUnyield: false,
    },
    mechanicsExtension: multiCharacterMechanicsExtension,
  },
};
