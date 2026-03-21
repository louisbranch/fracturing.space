import { ParticipantPortraitRail } from "../../shared/participant-portrait-rail/ParticipantPortraitRail";
import { sideChatRailParticipants } from "../../shared/view-models";
import type { SideChatParticipantRailProps } from "./contract";

export function SideChatParticipantRail({
  participants,
  viewerParticipantId,
  onParticipantInspect,
}: SideChatParticipantRailProps) {
  return (
    <ParticipantPortraitRail
      participants={sideChatRailParticipants(participants)}
      viewerParticipantId={viewerParticipantId}
      ariaLabel="Side chat participants"
      onParticipantInspect={onParticipantInspect}
    />
  );
}
