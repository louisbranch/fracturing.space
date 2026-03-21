import {
  characterAvatarPreviewAssets,
  participantAvatarPreviewAssets,
} from "../../../../storybook/preview-assets/fixtures";
import type {
  OnStageCharacterSummary,
  OnStageParticipant,
  OnStageSlot,
  OnStageState,
} from "./contract";

const [rheaAvatar, brynAvatar, guideAvatar] = participantAvatarPreviewAssets;
const [ariaAvatar, corinAvatar, sableAvatar, miraAvatar, rowanAvatar] =
  characterAvatarPreviewAssets;

export const onStageCharacterCatalog = {
  aria: { id: "char-aria", name: "Aria", avatarUrl: ariaAvatar?.imageUrl },
  corin: { id: "char-corin", name: "Corin", avatarUrl: corinAvatar?.imageUrl },
  sable: { id: "char-sable", name: "Sable", avatarUrl: sableAvatar?.imageUrl },
  mira: { id: "char-mira", name: "Mira", avatarUrl: miraAvatar?.imageUrl },
  rowan: { id: "char-rowan", name: "Rowan", avatarUrl: rowanAvatar?.imageUrl },
} satisfies Record<string, OnStageCharacterSummary>;

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

function slot(
  input: Omit<OnStageSlot, "id"> & { id?: string },
): OnStageSlot {
  return {
    id: input.id ?? `${input.participantId}-${input.characters.map((character) => character.id).join("-")}`,
    ...input,
  };
}

const baseScene = {
  sceneName: "Sealed Vault",
  sceneDescription:
    "A humming ward seals the old vault while a hairline seam catches the lantern light.",
  gmOutputText:
    "The vault door hums with old warding magic, and the seam flashes whenever a hand drifts too close.",
};
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
    ...baseScene,
    mode: "waiting-on-gm",
    aiStatus: "idle",
    viewerParticipantId: "p-rhea",
    actingParticipantIds: [],
    actingCharacterNames: [],
    gmAuthorityParticipantId: "p-guide",
    participants: onStageParticipants.map((participant) => ({
      ...participant,
      railStatus: "waiting",
    })),
    slots: [],
    viewerControls: {
      canSubmit: false,
      canSubmitAndYield: false,
      canYield: false,
      canUnyield: false,
      disabledReason: "Waiting for the GM to frame the next beat.",
    },
    mechanicsExtension: defaultMechanicsExtension,
  },
  actingEmpty: {
    ...baseScene,
    mode: "acting",
    aiStatus: "idle",
    frameText:
      "The ward crackles when either of you nears the seam. What do you do before the alarm wakes the keep?",
    viewerParticipantId: "p-rhea",
    actingParticipantIds: ["p-rhea", "p-bryn"],
    actingCharacterNames: ["Aria", "Corin"],
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
    viewerControls: {
      canSubmit: true,
      canSubmitAndYield: true,
      canYield: true,
      canUnyield: false,
    },
    mechanicsExtension: defaultMechanicsExtension,
  },
  viewerPosted: {
    ...baseScene,
    mode: "acting",
    aiStatus: "idle",
    frameText:
      "The ward crackles when either of you nears the seam. What do you do before the alarm wakes the keep?",
    viewerParticipantId: "p-rhea",
    actingParticipantIds: ["p-rhea", "p-bryn"],
    actingCharacterNames: ["Aria", "Corin"],
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
    viewerControls: {
      canSubmit: true,
      canSubmitAndYield: true,
      canYield: true,
      canUnyield: false,
    },
    mechanicsExtension: defaultMechanicsExtension,
  },
  yieldedWaiting: {
    ...baseScene,
    mode: "yielded-waiting",
    aiStatus: "idle",
    frameText:
      "The ward crackles when either of you nears the seam. What do you do before the alarm wakes the keep?",
    viewerParticipantId: "p-rhea",
    actingParticipantIds: ["p-rhea", "p-bryn"],
    actingCharacterNames: ["Aria", "Corin"],
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
    ...baseScene,
    mode: "changes-requested",
    aiStatus: "idle",
    frameText:
      "The vault seam opens a fraction, but the ward snaps toward the tool. Rhea, tighten the action and try again.",
    viewerParticipantId: "p-rhea",
    actingParticipantIds: ["p-rhea"],
    actingCharacterNames: ["Aria"],
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
    viewerControls: {
      canSubmit: true,
      canSubmitAndYield: true,
      canYield: false,
      canUnyield: false,
    },
    mechanicsExtension: defaultMechanicsExtension,
  },
  oocBlocked: {
    ...baseScene,
    mode: "ooc-blocked",
    aiStatus: "idle",
    frameText:
      "The ward crackles when either of you nears the seam. What do you do before the alarm wakes the keep?",
    oocReason: "Clarify whether tools touching the seam trigger the ward.",
    viewerParticipantId: "p-rhea",
    actingParticipantIds: ["p-rhea", "p-bryn"],
    actingCharacterNames: ["Aria", "Corin"],
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
    ...baseScene,
    mode: "waiting-on-gm",
    aiStatus: "running",
    viewerParticipantId: "p-rhea",
    actingParticipantIds: [],
    actingCharacterNames: [],
    gmAuthorityParticipantId: "p-guide",
    participants: onStageParticipants.map((participant) => ({
      ...participant,
      railStatus: "waiting",
    })),
    slots: [],
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
    ...baseScene,
    mode: "waiting-on-gm",
    aiStatus: "failed",
    viewerParticipantId: "p-rhea",
    actingParticipantIds: [],
    actingCharacterNames: [],
    gmAuthorityParticipantId: "p-guide",
    participants: onStageParticipants.map((participant) => ({
      ...participant,
      railStatus: "waiting",
    })),
    slots: [],
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
    sceneName: "Moonlit Courtyard",
    sceneDescription:
      "A fountain masks quiet footfalls while four shadows move between the arches.",
    gmOutputText:
      "The courtyard is quiet for now, but the keep windows are still lit above the fountain.",
    mode: "acting",
    aiStatus: "idle",
    frameText:
      "Your whole crew is in position. How do they move through the courtyard before the patrol turns back?",
    viewerParticipantId: "p-rhea",
    actingParticipantIds: ["p-rhea"],
    actingCharacterNames: ["Aria", "Sable", "Mira", "Rowan"],
    gmAuthorityParticipantId: "p-guide",
    participants: [
      {
        ...onStageParticipants[0],
        characters: [
          onStageCharacterCatalog.aria,
          onStageCharacterCatalog.sable,
          onStageCharacterCatalog.mira,
          onStageCharacterCatalog.rowan,
        ],
        railStatus: "active",
      },
      { ...onStageParticipants[2], railStatus: "waiting" },
    ],
    slots: [
      slot({
        participantId: "p-rhea",
        characters: [
          onStageCharacterCatalog.aria,
          onStageCharacterCatalog.sable,
          onStageCharacterCatalog.mira,
          onStageCharacterCatalog.rowan,
        ],
        body: "Aria watches the archway while Sable crosses to the fountain and Mira guides Rowan through the blind side of the lantern light.",
        updatedAt: "2026-03-19T20:48:00Z",
        yielded: false,
        reviewState: "open",
      }),
    ],
    viewerControls: {
      canSubmit: true,
      canSubmitAndYield: true,
      canYield: true,
      canUnyield: false,
    },
    mechanicsExtension: multiCharacterMechanicsExtension,
  },
};
