import type { PlayerHUDStatusBadge } from "../../shared/view-models";

export type OnStageSceneCardProps = {
  sceneName: string;
  sceneDescription?: string;
  gmOutputText?: string;
  frameText?: string;
  actingCharacterNames: string[];
  status: PlayerHUDStatusBadge;
};
