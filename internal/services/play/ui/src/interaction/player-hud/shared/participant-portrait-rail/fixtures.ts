import { participantAvatarPreviewAssets } from "../../../../storybook/preview-assets/fixtures";
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
      status: "idle",
    },
    {
      id: "p-bryn",
      name: "Bryn",
      roleLabel: "PLAYER",
      avatarUrl: otherParticipantAvatar?.imageUrl,
      status: "typing",
    },
    {
      id: "p-guide",
      name: "Guide",
      roleLabel: "GM",
      avatarUrl: gmAvatar?.imageUrl,
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
      status: "ready",
    },
    {
      id: "p-bryn",
      name: "Bryn",
      roleLabel: "PLAYER",
      avatarUrl: otherParticipantAvatar?.imageUrl,
      status: "ready",
    },
    {
      id: "p-guide",
      name: "Guide",
      roleLabel: "GM",
      avatarUrl: gmAvatar?.imageUrl,
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
      status: "active",
    },
    {
      id: "p-bryn",
      name: "Bryn",
      roleLabel: "PLAYER",
      avatarUrl: otherParticipantAvatar?.imageUrl,
      status: "yielded",
    },
    {
      id: "p-guide",
      name: "Guide",
      roleLabel: "GM",
      avatarUrl: gmAvatar?.imageUrl,
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
      status: "changes-requested",
    },
    {
      id: "p-bryn",
      name: "Bryn",
      roleLabel: "PLAYER",
      avatarUrl: otherParticipantAvatar?.imageUrl,
      status: "idle",
    },
    {
      id: "p-guide",
      name: "Guide",
      roleLabel: "GM",
      avatarUrl: gmAvatar?.imageUrl,
      status: "idle",
      ownsGMAuthority: true,
    },
  ],
};
