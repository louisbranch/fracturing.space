import { ParticipantPortraitRail } from "../../shared/participant-portrait-rail/ParticipantPortraitRail";
import { onStageRailParticipants } from "../../shared/view-models";
import type { OnStageParticipantRailProps } from "./contract";

export function OnStageParticipantRail({
  participants,
  viewerParticipantId,
  ariaLabel = "On-stage participants",
  onParticipantInspect,
}: OnStageParticipantRailProps) {
  return (
    <ParticipantPortraitRail
      participants={onStageRailParticipants(participants)}
      viewerParticipantId={viewerParticipantId}
      ariaLabel={ariaLabel}
      onParticipantInspect={onParticipantInspect}
    />
  );
}
