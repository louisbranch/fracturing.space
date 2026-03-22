import { ParticipantPortraitRail } from "../../shared/participant-portrait-rail/ParticipantPortraitRail";
import { backstageRailParticipants } from "../../shared/view-models";
import type { BackstageParticipantRailProps } from "./contract";

export function BackstageParticipantRail({
  participants,
  viewerParticipantId,
  gmAuthorityParticipantId,
  aiOwnerParticipantId,
  aiStatus,
  ariaLabel = "Backstage participants",
  onParticipantInspect,
}: BackstageParticipantRailProps) {
  return (
    <ParticipantPortraitRail
      participants={backstageRailParticipants({
        participants,
        gmAuthorityParticipantId,
      }, {
        ownerParticipantId: aiOwnerParticipantId,
        status: aiStatus,
      })}
      viewerParticipantId={viewerParticipantId}
      ariaLabel={ariaLabel}
      onParticipantInspect={onParticipantInspect}
    />
  );
}
