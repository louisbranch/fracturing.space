import type { BackstageParticipant } from "../shared/contract";
import { ParticipantPortraitRail } from "../../shared/participant-portrait-rail/ParticipantPortraitRail";
import type { BackstageParticipantRailProps } from "./contract";

function mapStatus(participant: BackstageParticipant): "idle" | "typing" | "ready" {
  if (participant.typing) {
    return "typing";
  }
  if (participant.readyToResume) {
    return "ready";
  }
  return "idle";
}

export function BackstageParticipantRail({
  participants,
  viewerParticipantId,
  gmAuthorityParticipantId,
  ariaLabel = "Backstage participants",
}: BackstageParticipantRailProps) {
  return (
    <ParticipantPortraitRail
      participants={participants.map((participant) => ({
        id: participant.id,
        name: participant.name,
        avatarUrl: participant.avatarUrl,
        roleLabel: participant.role.toUpperCase(),
        status: mapStatus(participant),
        ownsGMAuthority: participant.id === gmAuthorityParticipantId,
      }))}
      viewerParticipantId={viewerParticipantId}
      ariaLabel={ariaLabel}
    />
  );
}
