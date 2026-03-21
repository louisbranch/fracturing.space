import { participantAvatarPreviewAssets } from "../../../../storybook/preview-assets/fixtures";
import { playerHUDCharacterCatalog } from "../character-inspection-fixtures";
import type { ParticipantPortraitRailParticipant } from "./contract";

const [viewerAvatar, otherParticipantAvatar, gmAvatar] = participantAvatarPreviewAssets;

export const participantPortraitRailFixtures: Record<
  "typing" | "ready" | "active" | "changesRequested",
  ParticipantPortraitRailParticipant[]
> = {
  typing: [
    {
      id: "p-rhea",
      name: "Rhea",
      roleLabel: "PLAYER",
      avatarUrl: viewerAvatar?.imageUrl,
      characters: [playerHUDCharacterCatalog.aria, playerHUDCharacterCatalog.mira],
      status: "idle",
    },
    {
      id: "p-bryn",
      name: "Bryn",
      roleLabel: "PLAYER",
      avatarUrl: otherParticipantAvatar?.imageUrl,
      characters: [playerHUDCharacterCatalog.corin],
      status: "typing",
    },
    {
      id: "p-guide",
      name: "Guide",
      roleLabel: "GM",
      avatarUrl: gmAvatar?.imageUrl,
      characters: [],
      status: "idle",
      ownsGMAuthority: true,
    },
  ],
  ready: [
    {
      id: "p-rhea",
      name: "Rhea",
      roleLabel: "PLAYER",
      avatarUrl: viewerAvatar?.imageUrl,
      characters: [playerHUDCharacterCatalog.aria, playerHUDCharacterCatalog.mira],
      status: "ready",
    },
    {
      id: "p-bryn",
      name: "Bryn",
      roleLabel: "PLAYER",
      avatarUrl: otherParticipantAvatar?.imageUrl,
      characters: [playerHUDCharacterCatalog.corin],
      status: "ready",
    },
    {
      id: "p-guide",
      name: "Guide",
      roleLabel: "GM",
      avatarUrl: gmAvatar?.imageUrl,
      characters: [],
      status: "idle",
      ownsGMAuthority: true,
    },
  ],
  active: [
    {
      id: "p-rhea",
      name: "Rhea",
      roleLabel: "PLAYER",
      avatarUrl: viewerAvatar?.imageUrl,
      characters: [playerHUDCharacterCatalog.aria, playerHUDCharacterCatalog.mira],
      status: "active",
    },
    {
      id: "p-bryn",
      name: "Bryn",
      roleLabel: "PLAYER",
      avatarUrl: otherParticipantAvatar?.imageUrl,
      characters: [playerHUDCharacterCatalog.corin],
      status: "yielded",
    },
    {
      id: "p-guide",
      name: "Guide",
      roleLabel: "GM",
      avatarUrl: gmAvatar?.imageUrl,
      characters: [],
      status: "idle",
      ownsGMAuthority: true,
    },
  ],
  changesRequested: [
    {
      id: "p-rhea",
      name: "Rhea",
      roleLabel: "PLAYER",
      avatarUrl: viewerAvatar?.imageUrl,
      characters: [playerHUDCharacterCatalog.aria, playerHUDCharacterCatalog.mira],
      status: "changes-requested",
    },
    {
      id: "p-bryn",
      name: "Bryn",
      roleLabel: "PLAYER",
      avatarUrl: otherParticipantAvatar?.imageUrl,
      characters: [playerHUDCharacterCatalog.corin],
      status: "idle",
    },
    {
      id: "p-guide",
      name: "Guide",
      roleLabel: "GM",
      avatarUrl: gmAvatar?.imageUrl,
      characters: [],
      status: "idle",
      ownsGMAuthority: true,
    },
  ],
};
