import { ParticipantPortraitRail } from "../../shared/participant-portrait-rail/ParticipantPortraitRail";
import type { OnStageParticipantRailProps } from "./contract";

export function OnStageParticipantRail({
  participants,
  viewerParticipantId,
  ariaLabel = "On-stage participants",
}: OnStageParticipantRailProps) {
  return (
    <ParticipantPortraitRail
      participants={participants.map((participant) => ({
        id: participant.id,
        name: participant.name,
        avatarUrl: participant.avatarUrl,
        roleLabel: participant.role.toUpperCase(),
        status: participant.railStatus === "waiting" ? "idle" : participant.railStatus,
        ownsGMAuthority: participant.ownsGMAuthority,
      }))}
      viewerParticipantId={viewerParticipantId}
      ariaLabel={ariaLabel}
    />
  );
}
