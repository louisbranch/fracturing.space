import { participantAvatarPreviewAssets } from "../../../../storybook/preview-assets/fixtures";
import {
  playerHUDCharacterCatalog,
  playerHUDCharacterInspectionCatalog,
} from "../../shared/character-inspection-fixtures";
import type { BackstageState, BackstageParticipant } from "./contract";

const [viewerAvatar, otherPlayerAvatar, gmAvatar] = participantAvatarPreviewAssets;

export const backstageParticipants: BackstageParticipant[] = [
  {
    id: "p-rhea",
    name: "Rhea",
    role: "player",
    avatarUrl: viewerAvatar?.imageUrl,
    characters: [
      playerHUDCharacterCatalog.aria,
      playerHUDCharacterCatalog.sable,
      playerHUDCharacterCatalog.mira,
      playerHUDCharacterCatalog.rowan,
    ],
    readyToResume: false,
  },
  {
    id: "p-bryn",
    name: "Bryn",
    role: "player",
    avatarUrl: otherPlayerAvatar?.imageUrl,
    characters: [playerHUDCharacterCatalog.corin],
    readyToResume: false,
  },
  {
    id: "p-guide",
    name: "Guide",
    role: "gm",
    avatarUrl: gmAvatar?.imageUrl,
    characters: [],
    readyToResume: false,
  },
];

const oocReason = "Clarify how the ward reacts to tools touching the seam.";
const sceneName = "Sealed Vault";
const pausedPromptText = "The ward crackles when either of you nears the seam. What do you do?";
const gmAuthorityParticipantId = "p-guide";

export const backstageFixtureCatalog: Record<
  "dormant" | "openEmpty" | "openDiscussion" | "viewerReady" | "waitingOnGM",
  BackstageState
> = {
  dormant: {
    mode: "dormant",
    sceneName,
    pausedPromptText,
    gmAuthorityParticipantId,
    resumeState: "inactive",
    viewerParticipantId: "p-rhea",
    participants: backstageParticipants,
    messages: [],
    characterInspectionCatalog: playerHUDCharacterInspectionCatalog,
  },
  openEmpty: {
    mode: "open",
    sceneName,
    pausedPromptText,
    reason: oocReason,
    gmAuthorityParticipantId,
    resumeState: "collecting-ready",
    viewerParticipantId: "p-rhea",
    participants: backstageParticipants,
    messages: [],
    characterInspectionCatalog: playerHUDCharacterInspectionCatalog,
  },
  openDiscussion: {
    mode: "open",
    sceneName,
    pausedPromptText,
    reason: oocReason,
    gmAuthorityParticipantId,
    resumeState: "collecting-ready",
    viewerParticipantId: "p-rhea",
    participants: [
      backstageParticipants[0],
      { ...backstageParticipants[1], typing: true },
      backstageParticipants[2],
    ],
    messages: [
      {
        id: "ooc-1",
        participantId: "p-rhea",
        body: "Does the ward react to metal touching the seam or only skin?",
        sentAt: "2026-03-19T19:30:00Z",
      },
      {
        id: "ooc-2",
        participantId: "p-guide",
        body: "It reacts to contact with the seam itself, not the material.",
        sentAt: "2026-03-19T19:31:00Z",
      },
      {
        id: "ooc-3",
        participantId: "p-bryn",
        body: "Then Bryn can coach from a step back while Rhea handles the tool.",
        sentAt: "2026-03-19T19:32:00Z",
      },
    ],
    characterInspectionCatalog: playerHUDCharacterInspectionCatalog,
  },
  viewerReady: {
    mode: "open",
    sceneName,
    pausedPromptText,
    reason: oocReason,
    gmAuthorityParticipantId,
    resumeState: "collecting-ready",
    viewerParticipantId: "p-rhea",
    participants: [
      { ...backstageParticipants[0], readyToResume: true },
      backstageParticipants[1],
      backstageParticipants[2],
    ],
    messages: [
      {
        id: "ooc-1",
        participantId: "p-guide",
        body: "The ward reacts to contact with the seam itself, not the material.",
        sentAt: "2026-03-19T19:31:00Z",
      },
    ],
    characterInspectionCatalog: playerHUDCharacterInspectionCatalog,
  },
  waitingOnGM: {
    mode: "open",
    sceneName,
    pausedPromptText,
    reason: oocReason,
    gmAuthorityParticipantId,
    resumeState: "waiting-on-gm",
    viewerParticipantId: "p-rhea",
    participants: [
      { ...backstageParticipants[0], readyToResume: true },
      { ...backstageParticipants[1], readyToResume: true },
      backstageParticipants[2],
    ],
    messages: [
      {
        id: "ooc-1",
        participantId: "p-rhea",
        body: "I’m ready to move back on-stage.",
        sentAt: "2026-03-19T19:34:00Z",
      },
      {
        id: "ooc-2",
        participantId: "p-bryn",
        body: "Same here. Waiting on the next prompt.",
        sentAt: "2026-03-19T19:35:00Z",
      },
    ],
    characterInspectionCatalog: playerHUDCharacterInspectionCatalog,
  },
};
