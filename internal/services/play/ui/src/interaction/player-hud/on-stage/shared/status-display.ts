import type { OnStageAIStatus, OnStageMode } from "./contract";

export type OnStageStatusDisplay = {
  badgeClassName: string;
  badgeLabel: string;
  message: string;
};

export function onStageStatusDisplay(input: {
  mode: OnStageMode;
  aiStatus: OnStageAIStatus;
  disabledReason?: string;
  oocReason?: string;
}): OnStageStatusDisplay {
  if (input.mode === "ooc-blocked") {
    return {
      badgeClassName: "badge-warning badge-soft",
      badgeLabel: "OOC Open",
      message: input.oocReason
        ? `Backstage is open for a rules pause: ${input.oocReason}`
        : "Backstage OOC is open. Resolve the ruling there before acting on-stage.",
    };
  }

  if (input.mode === "changes-requested") {
    return {
      badgeClassName: "badge-warning badge-soft",
      badgeLabel: "Changes Requested",
      message: "GM asked you to tighten or revise your committed action.",
    };
  }

  if (input.mode === "yielded-waiting") {
    return {
      badgeClassName: "badge-secondary badge-soft",
      badgeLabel: "Yielded",
      message: input.disabledReason ?? "You have already yielded. Unyield if you need to revise before the beat closes.",
    };
  }

  if (input.mode === "acting") {
    return {
      badgeClassName: "badge-primary badge-soft",
      badgeLabel: "Your Beat",
      message: "Commit the next action for your character and yield when you are ready.",
    };
  }

  if (input.aiStatus === "running" || input.aiStatus === "queued") {
    return {
      badgeClassName: "badge-info badge-soft",
      badgeLabel: "AI Thinking",
      message: "The next beat is being framed. Hold position until the scene opens again.",
    };
  }

  if (input.aiStatus === "failed") {
    return {
      badgeClassName: "badge-error badge-soft",
      badgeLabel: "GM Delayed",
      message: "The next beat is delayed while GM authority reorients.",
    };
  }

  return {
    badgeClassName: "badge-ghost",
    badgeLabel: "Waiting",
    message: input.disabledReason ?? "Waiting for the GM to frame the next beat.",
  };
}
