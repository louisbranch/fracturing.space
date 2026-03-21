import { participantAvatarPreviewAssets } from "../../../../storybook/preview-assets/fixtures";
import type { ParticipantPortraitRailParticipant } from "./contract";

const [viewerAvatar, otherParticipantAvatar, gmAvatar] = participantAvatarPreviewAssets;

export const participantPortraitRailFixtures: Record<
  "typing" | "ready",
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
};
