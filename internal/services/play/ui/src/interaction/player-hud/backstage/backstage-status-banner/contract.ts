import type { BackstageMode, BackstageResumeState } from "../shared/contract";

export type BackstageStatusBannerProps = {
  mode: BackstageMode;
  resumeState: BackstageResumeState;
  viewerReady: boolean;
  onViewerReadyToggle: () => void;
};
