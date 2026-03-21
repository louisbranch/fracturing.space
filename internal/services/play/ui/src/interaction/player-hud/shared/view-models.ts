import type { BackstageParticipant, BackstageState } from "../backstage/shared/contract";
import type { OnStageParticipant, OnStageState } from "../on-stage/shared/contract";
import type { ParticipantPortraitRailParticipant, ParticipantPortraitStatus } from "./participant-portrait-rail/contract";
import type { SideChatParticipant } from "./contract";

// PlayerHUDStatusBadge keeps panel and card components focused on rendering a
// single badge view model instead of reassembling label/class/tooltip triplets.
export type PlayerHUDStatusBadge = {
  className: string;
  indicator?: "none" | "loading-bars";
  label: string;
  tooltip: string;
};

function roleLabel(role: "player" | "gm"): string {
  return role.toUpperCase();
}

function backstageParticipantStatus(participant: BackstageParticipant): ParticipantPortraitStatus {
  if (participant.typing) {
    return "typing";
  }
  if (participant.readyToResume) {
    return "ready";
  }
  return "idle";
}

function onStageParticipantStatus(participant: OnStageParticipant): ParticipantPortraitStatus {
  return participant.railStatus === "waiting" ? "idle" : participant.railStatus;
}

export function backstageStatusBadge(
  state: Pick<BackstageState, "mode" | "resumeState" | "viewerParticipantId" | "participants">,
): PlayerHUDStatusBadge {
  const viewer = state.participants.find((participant) => participant.id === state.viewerParticipantId);
  const viewerReady = Boolean(viewer?.readyToResume);

  if (state.mode !== "open") {
    return {
      className: "badge-ghost",
      label: "Backstage Idle",
      indicator: "none",
      tooltip: "OOC is closed.",
    };
  }

  if (state.resumeState === "waiting-on-gm") {
    return {
      className: "badge-info badge-soft",
      label: "Waiting on GM",
      indicator: "loading-bars",
      tooltip: "All players are ready. Waiting for the GM to resume on-stage play.",
    };
  }

  if (viewerReady) {
    return {
      className: "badge-success badge-soft",
      label: "Ready",
      indicator: "none",
      tooltip: "You are ready. Clear Ready if you need to continue the OOC discussion.",
    };
  }

  return {
    className: "badge-warning badge-soft",
    label: "OOC Open",
    indicator: "none",
    tooltip: "Awaiting player readiness.",
  };
}

export function onStageStatusBadge(
  state: Pick<OnStageState, "mode" | "aiStatus" | "oocReason" | "viewerControls">,
): PlayerHUDStatusBadge {
  if (state.mode === "ooc-blocked") {
    return {
      className: "badge-warning badge-soft",
      label: "OOC Open",
      indicator: "none",
      tooltip: state.oocReason
        ? `Backstage is open for a rules pause: ${state.oocReason}`
        : "Backstage OOC is open. Resolve the ruling there before acting on-stage.",
    };
  }

  if (state.mode === "changes-requested") {
    return {
      className: "badge-warning badge-soft",
      label: "Changes Requested",
      indicator: "none",
      tooltip: "GM asked you to tighten or revise your committed action.",
    };
  }

  if (state.mode === "yielded-waiting") {
    return {
      className: "badge-secondary badge-soft",
      label: "Yielded",
      indicator: "none",
      tooltip:
        state.viewerControls.disabledReason
        ?? "You have already yielded. Unyield if you need to revise before the beat closes.",
    };
  }

  if (state.mode === "acting") {
    return {
      className: "badge-primary badge-soft",
      label: "Your Beat",
      indicator: "none",
      tooltip: "Commit the next action for your character and yield when you are ready.",
    };
  }

  if (state.aiStatus === "running" || state.aiStatus === "queued") {
    return {
      className: "badge-info badge-soft",
      label: "AI Thinking",
      indicator: "loading-bars",
      tooltip: "The next beat is being framed. Hold position until the scene opens again.",
    };
  }

  if (state.aiStatus === "failed") {
    return {
      className: "badge-error badge-soft",
      label: "GM Delayed",
      indicator: "none",
      tooltip: "The next beat is delayed while GM authority reorients.",
    };
  }

  return {
    className: "badge-ghost",
    label: "Waiting",
    indicator: "loading-bars",
    tooltip: state.viewerControls.disabledReason ?? "Waiting for the GM to frame the next beat.",
  };
}

export function backstageRailParticipants(
  state: Pick<BackstageState, "participants" | "gmAuthorityParticipantId">,
): ParticipantPortraitRailParticipant[] {
  return state.participants.map((participant) => ({
    id: participant.id,
    name: participant.name,
    avatarUrl: participant.avatarUrl,
    characters: participant.characters,
    roleLabel: roleLabel(participant.role),
    status: backstageParticipantStatus(participant),
    ownsGMAuthority: participant.id === state.gmAuthorityParticipantId,
  }));
}

export function onStageRailParticipants(
  participants: OnStageParticipant[],
): ParticipantPortraitRailParticipant[] {
  return participants.map((participant) => ({
    id: participant.id,
    name: participant.name,
    avatarUrl: participant.avatarUrl,
    characters: participant.characters,
    roleLabel: roleLabel(participant.role),
    status: onStageParticipantStatus(participant),
    ownsGMAuthority: participant.ownsGMAuthority,
  }));
}

export function sideChatRailParticipants(
  participants: SideChatParticipant[],
): ParticipantPortraitRailParticipant[] {
  return participants.map((participant) => ({
    id: participant.id,
    name: participant.name,
    avatarUrl: participant.avatarUrl,
    characters: participant.characters,
    roleLabel: roleLabel(participant.role),
    status: participant.typing ? "typing" : "idle",
  }));
}
