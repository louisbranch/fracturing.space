import { describe, expect, it } from "vitest";
import type { PlayBootstrap, PlayRoomSnapshot } from "./protocol";
import {
  createPlayShellViewModel,
  createSystemRenderViewModel,
  formatSessionLabel,
} from "./view_models";

const bootstrap: PlayBootstrap = {
  campaign_id: "camp-1",
  viewer: { participant_id: "player-1", name: "Avery", role: "PLAYER" },
  system: { id: "daggerheart", version: "1.0.0", name: "Daggerheart" },
  interaction_state: {
    campaign_id: "camp-1",
    campaign_name: "Guildhouse",
    active_session: { session_id: "sess-1", name: "Opening" },
    active_scene: {
      scene_id: "scene-1",
      session_id: "sess-1",
      name: "Town Gate",
      description: "A tense arrival.",
      characters: [{ character_id: "char-1", name: "Morrow", owner_participant_id: "player-1" }],
    },
    player_phase: {
      phase_id: "phase-1",
      status: 2,
      frame_text: "",
      acting_character_ids: [],
      acting_participant_ids: [],
      slots: [
        {
          participant_id: "player-1",
          summary_text: "Holding position.",
          character_ids: [],
          yielded: false,
          review_status: 0,
          review_reason: "",
          review_character_ids: [],
        },
      ],
    },
    ooc: {
      open: true,
      posts: [],
      ready_to_resume_participant_ids: ["player-1", "player-2"],
    },
    gm_authority_participant_id: "gm-1",
    ai_turn: {
      status: 2,
      turn_token: "",
      owner_participant_id: "",
      source_event_type: "",
      source_scene_id: "",
      source_phase_id: "",
      last_error: "",
    },
    viewer: { participant_id: "player-1", name: "Avery", role: "PLAYER" },
  },
  chat: {
    session_id: "sess-1",
    latest_sequence_id: 10,
    messages: [],
    history_url: "/api/campaigns/camp-1/chat/history",
  },
  realtime: { url: "/ws/play", protocol_version: 1 },
};

const snapshot: PlayRoomSnapshot = {
  interaction_state: bootstrap.interaction_state,
  latest_game_sequence: 3,
  chat: bootstrap.chat,
};

describe("view models", () => {
  it("formats session labels for missing and unnamed sessions", () => {
    expect(formatSessionLabel()).toBe("No active session");
    expect(formatSessionLabel({ session_id: "sess-1", name: "" })).toBe("Untitled session");
    expect(formatSessionLabel({ session_id: "sess-1", name: "The Old Man" })).toBe("The Old Man");
  });

  it("builds the shell view model from snapshot state", () => {
    const view = createPlayShellViewModel(bootstrap, snapshot, true);

    expect(view.campaignName).toBe("Guildhouse");
    expect(view.viewerName).toBe("Avery");
    expect(view.sessionLabel).toBe("Opening");
    expect(view.systemLabel).toBe("Daggerheart");
    expect(view.connectedLabel).toBe("Connected");
  });

  it("builds stable renderer labels from protocol state", () => {
    const view = createSystemRenderViewModel(snapshot);

    expect(view.scenePhaseLabel).toBe("Players acting");
    expect(view.oocLabel).toBe("OOC paused · 2 ready");
    expect(view.aiTurnLabel).toBe("Queued");
    expect(view.scene.title).toBe("Town Gate");
    expect(view.scene.characters).toEqual([{ id: "char-1", name: "Morrow" }]);
    expect(view.slots).toEqual([
      {
        key: "player-1-slot",
        participantLabel: "player-1",
        summaryText: "Holding position.",
        statusLabel: "Pending",
      },
    ]);
  });
});
