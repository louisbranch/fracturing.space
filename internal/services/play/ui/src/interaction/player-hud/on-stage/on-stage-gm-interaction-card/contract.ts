import type { PlayerHUDStatusBadge } from "../../shared/view-models";
import type { OnStageGMInteraction } from "../shared/contract";

export type OnStageGMInteractionCardProps = {
  currentInteraction?: OnStageGMInteraction;
  interactionHistory: OnStageGMInteraction[];
  currentStatus: PlayerHUDStatusBadge;
};
