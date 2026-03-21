import { describe, expect, it } from "vitest";
import { backstageFixtureCatalog } from "../backstage/shared/fixtures";
import { onStageFixtureCatalog } from "../on-stage/shared/fixtures";
import { sideChatState } from "./fixtures";
import {
  backstageRailParticipants,
  backstageStatusBadge,
  onStageRailParticipants,
  onStageStatusBadge,
  sideChatRailParticipants,
} from "./view-models";

describe("player HUD view models", () => {
  it("derives backstage status badges from the full shared state", () => {
    expect(backstageStatusBadge(backstageFixtureCatalog.openDiscussion)).toEqual({
      className: "badge-warning badge-soft",
      indicator: "none",
      label: "OOC Open",
      tooltip: "Awaiting player readiness.",
    });

    expect(backstageStatusBadge(backstageFixtureCatalog.waitingOnGM)).toEqual({
      className: "badge-info badge-soft",
      indicator: "loading-bars",
      label: "Waiting on GM",
      tooltip: "All players are ready. Waiting for the GM to resume on-stage play.",
    });
  });

  it("derives on-stage status badges from the full shared state", () => {
    expect(onStageStatusBadge(onStageFixtureCatalog.viewerPosted)).toEqual({
      className: "badge-primary badge-soft",
      indicator: "none",
      label: "Your Beat",
      tooltip: "Commit the next action for your character and yield when you are ready.",
    });

    expect(onStageStatusBadge(onStageFixtureCatalog.oocBlocked)).toEqual({
      className: "badge-warning badge-soft",
      indicator: "none",
      label: "OOC Open",
      tooltip:
        "Backstage is open for a rules pause: Clarify whether tools touching the seam trigger the ward.",
    });

    expect(onStageStatusBadge(onStageFixtureCatalog.waitingOnGM)).toEqual({
      className: "badge-ghost",
      indicator: "loading-bars",
      label: "Waiting",
      tooltip: "Waiting for the GM to open the next beat.",
    });

    expect(onStageStatusBadge(onStageFixtureCatalog.aiThinking)).toEqual({
      className: "badge-info badge-soft",
      indicator: "loading-bars",
      label: "AI Thinking",
      tooltip: "The next prompt is being prepared. Hold position until the scene opens again.",
    });

    expect(onStageStatusBadge(onStageFixtureCatalog.aiFailed)).toEqual({
      className: "badge-error badge-soft",
      indicator: "none",
      label: "GM Delayed",
      tooltip: "The next beat is delayed while GM authority reorients.",
    });
  });

  it("maps backstage, on-stage, and side-chat participants into portrait-rail view models", () => {
    expect(backstageRailParticipants(backstageFixtureCatalog.openDiscussion)).toMatchObject([
      {
        id: "p-rhea",
        roleLabel: "PLAYER",
        status: "idle",
      },
      {
        id: "p-bryn",
        roleLabel: "PLAYER",
        status: "typing",
      },
      {
        id: "p-guide",
        roleLabel: "GM",
        status: "idle",
        ownsGMAuthority: true,
      },
    ]);

    expect(onStageRailParticipants(onStageFixtureCatalog.yieldedWaiting.participants)).toMatchObject([
      {
        id: "p-rhea",
        roleLabel: "PLAYER",
        status: "yielded",
      },
      {
        id: "p-bryn",
        roleLabel: "PLAYER",
        status: "active",
      },
      {
        id: "p-guide",
        roleLabel: "GM",
        status: "idle",
        ownsGMAuthority: true,
      },
    ]);

    expect(sideChatRailParticipants(sideChatState.participants)).toMatchObject([
      {
        id: "p-rhea",
        roleLabel: "PLAYER",
        status: "idle",
      },
      {
        id: "p-bryn",
        roleLabel: "PLAYER",
        status: "typing",
      },
      {
        id: "p-guide",
        roleLabel: "GM",
        status: "idle",
      },
    ]);
  });
});
