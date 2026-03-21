import type { BackstageMode, BackstageResumeState } from "./contract";

export type BackstageStatusDisplay = {
  badgeClassName: string;
  badgeLabel: string;
  message: string;
};

export function backstageStatusDisplay(input: {
  mode: BackstageMode;
  resumeState: BackstageResumeState;
  viewerReady: boolean;
}): BackstageStatusDisplay {
  if (input.mode !== "open") {
    return {
      badgeClassName: "badge-ghost",
      badgeLabel: "Backstage Idle",
      message: "OOC is closed.",
    };
  }

  if (input.resumeState === "waiting-on-gm") {
    return {
      badgeClassName: "badge-info badge-soft",
      badgeLabel: "Waiting on GM",
      message: "All players are ready. Waiting for the GM to resume on-stage play.",
    };
  }

  if (input.viewerReady) {
    return {
      badgeClassName: "badge-success badge-soft",
      badgeLabel: "Ready",
      message: "You are ready. Clear Ready if you need to continue the OOC discussion.",
    };
  }

  return {
    badgeClassName: "badge-warning badge-soft",
    badgeLabel: "OOC Open",
    message: "Awaiting player readiness.",
  };
}
