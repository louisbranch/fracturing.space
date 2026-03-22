import { ParticipantPortraitRail } from "../../shared/participant-portrait-rail/ParticipantPortraitRail";
import { sideChatRailParticipants } from "../../shared/view-models";
import type { SideChatParticipantRailProps } from "./contract";

export function SideChatParticipantRail({
  participants,
  viewerParticipantId,
  aiOwnerParticipantId,
  aiStatus,
  onParticipantInspect,
}: SideChatParticipantRailProps) {
  return (
    <ParticipantPortraitRail
      participants={sideChatRailParticipants(participants, {
        ownerParticipantId: aiOwnerParticipantId,
        status: aiStatus,
      })}
      viewerParticipantId={viewerParticipantId}
      ariaLabel="Side chat participants"
      onParticipantInspect={onParticipantInspect}
    />
  );
}
