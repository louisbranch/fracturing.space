import { describe, expect, it } from "vitest";
import { deriveOnStageMode, mapToPlayerHUDState } from "./mapper";
import type { BootstrapResponse, WireOOCState, WirePlayerPhase } from "../api/types";

function minimalBootstrap(overrides?: Partial<BootstrapResponse>): BootstrapResponse {
  return {
    campaign_id: "c1",
    viewer: { participant_id: "p1", name: "Avery", role: "player" },
    system: { id: "daggerheart", version: "1.0", name: "Daggerheart" },
    interaction_state: {
      campaign_id: "c1",
      campaign_name: "The Guildhouse",
      viewer: { participant_id: "p1", name: "Avery", role: "player" },
      active_session: { session_id: "s1", name: "Session 1" },
      active_scene: {
        scene_id: "sc1",
        name: "The Tavern",
        characters: [{ character_id: "ch1", name: "Lark", owner_participant_id: "p1" }],
      },
    },
    participants: [
      { id: "p1", name: "Avery", role: "player", character_ids: ["ch1"] },
      { id: "p2", name: "GM Bran", role: "gm", character_ids: [] },
    ],
    character_inspection_catalog: null,
    chat: {
      session_id: "s1",
      latest_sequence_id: 0,
      messages: [],
      history_url: "/api/campaigns/c1/chat/history",
    },
    realtime: { url: "/realtime", protocol_version: 1 },
    ...overrides,
  };
}

describe("deriveOnStageMode", () => {
  it("returns ooc-blocked when OOC is open", () => {
    const ooc: WireOOCState = { open: true, posts: [], ready_to_resume_participant_ids: [] };
    expect(deriveOnStageMode(undefined, ooc, "p1")).toBe("ooc-blocked");
  });

  it("returns waiting-on-gm when no phase", () => {
    expect(deriveOnStageMode(undefined, undefined, "p1")).toBe("waiting-on-gm");
  });

  it("returns waiting-on-gm when phase status is gm", () => {
    const phase: WirePlayerPhase = {
      phase_id: "ph1",
      status: "gm",
      acting_character_ids: [],
      acting_participant_ids: [],
      slots: [],
    };
    expect(deriveOnStageMode(phase, undefined, "p1")).toBe("waiting-on-gm");
  });

  it("returns acting when viewer is in acting list", () => {
    const phase: WirePlayerPhase = {
      phase_id: "ph1",
      status: "players",
      acting_character_ids: ["ch1"],
      acting_participant_ids: ["p1"],
      slots: [{ participant_id: "p1", character_ids: [], yielded: false, review_character_ids: [] }],
    };
    expect(deriveOnStageMode(phase, undefined, "p1")).toBe("acting");
  });

  it("returns yielded-waiting when viewer slot is yielded", () => {
    const phase: WirePlayerPhase = {
      phase_id: "ph1",
      status: "players",
      acting_character_ids: [],
      acting_participant_ids: ["p1"],
      slots: [{ participant_id: "p1", character_ids: [], yielded: true, review_character_ids: [] }],
    };
    expect(deriveOnStageMode(phase, undefined, "p1")).toBe("yielded-waiting");
  });

  it("returns changes-requested when viewer slot has changes requested", () => {
    const phase: WirePlayerPhase = {
      phase_id: "ph1",
      status: "players",
      acting_character_ids: [],
      acting_participant_ids: ["p1"],
      slots: [
        {
          participant_id: "p1",
          character_ids: [],
          yielded: false,
          review_status: "changes_requested",
          review_character_ids: [],
        },
      ],
    };
    expect(deriveOnStageMode(phase, undefined, "p1")).toBe("changes-requested");
  });
});

describe("mapToPlayerHUDState", () => {
  it("maps bootstrap data to a complete HUD state", () => {
    const bootstrap = minimalBootstrap();
    const state = mapToPlayerHUDState(bootstrap, null, "connected", "on-stage", []);

    expect(state.activeTab).toBe("on-stage");
    expect(state.connectionState).toBe("connected");
    expect(state.onStage.sceneName).toBe("The Tavern");
    expect(state.onStage.viewerParticipantId).toBe("p1");
    expect(state.backstage.viewerParticipantId).toBe("p1");
    expect(state.sideChat.viewerParticipantId).toBe("p1");
    expect(state.campaignNavigation.returnHref).toBe("/app/campaigns/c1/game");
  });

  it("maps chat messages to side chat", () => {
    const bootstrap = minimalBootstrap();
    const messages = [
      {
        message_id: "m1",
        campaign_id: "c1",
        session_id: "s1",
        sequence_id: 1,
        sent_at: "2026-03-19T12:00:00Z",
        actor: { participant_id: "p1", name: "Avery" },
        body: "Hello!",
      },
    ];
    const state = mapToPlayerHUDState(bootstrap, null, "connected", "side-chat", messages);

    expect(state.sideChat.messages).toHaveLength(1);
    expect(state.sideChat.messages[0].body).toBe("Hello!");
    expect(state.sideChat.messages[0].participantId).toBe("p1");
  });

  it("maps participants with character references", () => {
    const bootstrap = minimalBootstrap();
    const state = mapToPlayerHUDState(bootstrap, null, "connected", "on-stage", []);

    expect(state.campaignNavigation.characterControllers).toHaveLength(2);
    const avery = state.campaignNavigation.characterControllers.find((c) => c.participantId === "p1");
    expect(avery?.isViewer).toBe(true);
    expect(avery?.characters).toHaveLength(1);
    expect(avery?.characters[0].id).toBe("ch1");
  });
});
