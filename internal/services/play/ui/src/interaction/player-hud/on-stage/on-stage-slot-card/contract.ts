import type { OnStageParticipant, OnStageSlot } from "../shared/contract";

export type OnStageSlotCardProps = {
  slot: OnStageSlot;
  participant: OnStageParticipant;
  isViewer: boolean;
};
