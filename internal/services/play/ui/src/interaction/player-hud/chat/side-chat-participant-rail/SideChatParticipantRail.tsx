import { ParticipantPortraitRail } from "../../shared/participant-portrait-rail/ParticipantPortraitRail";
import type { SideChatParticipantRailProps } from "./contract";

export function SideChatParticipantRail({
  participants,
  viewerParticipantId,
  onParticipantInspect,
}: SideChatParticipantRailProps) {
  return (
    <ParticipantPortraitRail
      participants={participants.map((participant) => ({
        id: participant.id,
        name: participant.name,
        roleLabel: participant.role.toUpperCase(),
        avatarUrl: participant.avatarUrl,
        characters: participant.characters,
        status: participant.typing ? "typing" : "idle",
      }))}
      viewerParticipantId={viewerParticipantId}
      ariaLabel="Side chat participants"
      onParticipantInspect={onParticipantInspect}
    />
  );
}
