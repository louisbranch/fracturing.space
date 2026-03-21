import { OnStageSlotCard } from "../on-stage-slot-card/OnStageSlotCard";
import type { OnStageSlotListProps } from "./contract";

export function OnStageSlotList({
  participants,
  slots,
  actingParticipantIds,
  viewerParticipantId,
  ariaLabel = "On-stage messages",
  onCharacterInspect,
}: OnStageSlotListProps) {
  const participantMap = new Map(participants.map((participant) => [participant.id, participant]));
  const slotMap = new Map(slots.map((slot) => [slot.participantId, slot]));
  const orderedParticipantIDs = actingParticipantIds.length > 0
    ? actingParticipantIds
    : slots.map((slot) => slot.participantId);

  if (orderedParticipantIDs.length === 0) {
    return (
      <section aria-label={ariaLabel} className="border-t border-base-300/70 px-3 py-3">
        <div className="rounded-box border border-dashed border-base-300/70 bg-base-200/30 px-3 py-4 text-center text-sm text-base-content/55">
          No active player slots yet.
        </div>
      </section>
    );
  }

  return (
    <section aria-label={ariaLabel} className="flex flex-col gap-2 px-3 py-3">
      {orderedParticipantIDs.map((participantId) => {
        const participant = participantMap.get(participantId);
        if (!participant) {
          return null;
        }

        const slot = slotMap.get(participantId) ?? {
          id: `${participantId}-empty`,
          participantId,
          characters: participant.characters,
          yielded: false,
          reviewState: "open" as const,
        };

        return (
          <OnStageSlotCard
            key={slot.id}
            slot={slot}
            participant={participant}
            isViewer={participantId === viewerParticipantId}
            onCharacterInspect={onCharacterInspect}
          />
        );
      })}
    </section>
  );
}
