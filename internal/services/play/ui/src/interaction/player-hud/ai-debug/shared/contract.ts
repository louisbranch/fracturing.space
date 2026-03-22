import type { WireAIDebugTurn, WireAIDebugTurnSummary } from "../../../../api/types";

export type AIDebugPanelState = {
  phase: "idle" | "loading" | "ready" | "error";
  turns: WireAIDebugTurnSummary[];
  expandedTurnId?: string;
  detailsByTurnId: Record<string, WireAIDebugTurn>;
  nextPageToken?: string;
  loadingTurnId?: string;
  errorMessage?: string;
};
