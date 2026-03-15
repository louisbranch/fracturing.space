import type { PlayAITurnData } from "../shared/contract";

export type AITurnStatusBannerProps = {
  aiTurn: PlayAITurnData;
  onRetry?: () => void;
};
