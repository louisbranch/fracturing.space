import type { PlayerHUDStatusBadge } from "../../shared/view-models";

export type BackstageContextCardProps = {
  sceneName?: string;
  pausedPromptText?: string;
  reason?: string;
  status: PlayerHUDStatusBadge;
};
