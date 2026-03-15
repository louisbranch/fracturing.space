import type { PlayPhaseData, PlayViewerRole } from "../shared/contract";

export type PhaseStatusBannerProps = {
  phase: PlayPhaseData;
  viewerName?: string;
  viewerRole?: PlayViewerRole;
};
