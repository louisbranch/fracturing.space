import type { OnStageParticipant, OnStageSlot } from "../shared/contract";

export type OnStageSlotListProps = {
  participants: OnStageParticipant[];
  slots: OnStageSlot[];
  actingParticipantIds: string[];
  viewerParticipantId: string;
  ariaLabel?: string;
  onCharacterInspect?: (participantId: string, characterId: string) => void;
};
