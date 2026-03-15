import { describe, expect, it } from "vitest";
import type { PlayBootstrap, PlayRoomSnapshot } from "./protocol";
import {
  applyHistoryFailure,
  applyHistoryLoaded,
  applyHistoryLoadStarted,
  applyRealtimeFrame,
  applyRealtimeStatus,
  createLoadedRuntimeState,
} from "./runtime_state";

const bootstrap: PlayBootstrap = {
  campaign_id: "camp-1",
  viewer: { participant_id: "player-1", name: "Avery", role: "PLAYER" },
  system: { id: "daggerheart", version: "1.0.0", name: "Daggerheart" },
  interaction_state: {
    campaign_id: "camp-1",
    campaign_name: "Guildhouse",
    active_session: { session_id: "sess-1", name: "Opening" },
    gm_authority_participant_id: "gm-1",
  },
  chat: {
    session_id: "sess-1",
    latest_sequence_id: 10,
    messages: [
      {
        message_id: "msg-10",
        campaign_id: "camp-1",
        session_id: "sess-1",
        sequence_id: 10,
        sent_at: "2026-03-14T12:01:00Z",
        actor: { participant_id: "gm-1", name: "GM" },
        body: "The scene opens.",
      },
    ],
    history_url: "/api/campaigns/camp-1/chat/history",
  },
  realtime: { url: "/ws/play", protocol_version: 1 },
};

const updatedSnapshot: PlayRoomSnapshot = {
  interaction_state: {
    ...bootstrap.interaction_state,
    campaign_name: "Guildhouse Updated",
  },
  latest_game_sequence: 14,
  chat: {
    session_id: "sess-1",
    latest_sequence_id: 11,
    messages: [],
    history_url: "/api/campaigns/camp-1/chat/history",
  },
};

describe("runtime state", () => {
  it("merges realtime snapshot updates into the loaded state", () => {
    const current = createLoadedRuntimeState({
      bootstrap,
      snapshot: {
        interaction_state: bootstrap.interaction_state,
        latest_game_sequence: 0,
        chat: bootstrap.chat,
      },
      messages: bootstrap.chat.messages,
    });

    const next = applyRealtimeFrame(current, {
      type: "play.interaction.updated",
      payload: updatedSnapshot,
    });

    expect(next.connected).toBe(true);
    expect(next.snapshot?.interaction_state.campaign_name).toBe("Guildhouse Updated");
    expect(next.snapshot?.chat.latest_sequence_id).toBe(11);
  });

  it("deduplicates and appends chat history pages", () => {
    const current = applyHistoryLoadStarted(
      createLoadedRuntimeState({
        bootstrap,
        snapshot: {
          interaction_state: bootstrap.interaction_state,
          latest_game_sequence: 0,
          chat: bootstrap.chat,
        },
        messages: bootstrap.chat.messages,
      }),
    );

    const next = applyHistoryLoaded(current, {
      messages: [
        {
          message_id: "msg-9",
          campaign_id: "camp-1",
          session_id: "sess-1",
          sequence_id: 9,
          sent_at: "2026-03-14T12:00:00Z",
          actor: { participant_id: "player-1", name: "Avery" },
          body: "Ready at the gate.",
        },
        bootstrap.chat.messages[0],
      ],
    });

    expect(next.loadingHistory).toBe(false);
    expect(next.messages.map((message) => message.sequence_id)).toEqual([9, 10]);
  });

  it("tracks typing, transport errors, and disconnect state transitions", () => {
    const loaded = createLoadedRuntimeState({
      bootstrap,
      snapshot: {
        interaction_state: bootstrap.interaction_state,
        latest_game_sequence: 0,
        chat: bootstrap.chat,
      },
      messages: bootstrap.chat.messages,
    });

    const typing = applyRealtimeFrame(loaded, {
      type: "play.chat.typing",
      payload: {
        session_id: "sess-1",
        participant_id: "player-1",
        name: "Avery",
        active: true,
      },
    });
    const errored = applyRealtimeFrame(typing, {
      type: "play.error",
      payload: { error: { code: "invalid_argument", message: "body is required" } },
    });
    const disconnected = applyRealtimeStatus(errored, {
      type: "disconnected",
      message: "server restart",
    });
    const clearedTyping = applyRealtimeFrame(disconnected, {
      type: "play.chat.typing",
      payload: {
        session_id: "sess-1",
        participant_id: "player-1",
        name: "Avery",
        active: false,
      },
    });

    expect(Object.keys(typing.chatTyping)).toEqual(["player-1"]);
    expect(errored.error).toBe("body is required");
    expect(disconnected.connected).toBe(false);
    expect(disconnected.error).toBe("server restart");
    expect(clearedTyping.chatTyping).toEqual({});
  });

  it("clears loadingHistory and preserves messages on history failure", () => {
    const current = applyHistoryLoadStarted(
      createLoadedRuntimeState({
        bootstrap,
        snapshot: {
          interaction_state: bootstrap.interaction_state,
          latest_game_sequence: 0,
          chat: bootstrap.chat,
        },
        messages: bootstrap.chat.messages,
      }),
    );

    const next = applyHistoryFailure(current, "history request failed: 500");

    expect(next.loadingHistory).toBe(false);
    expect(next.messages.map((message) => message.sequence_id)).toEqual([10]);
    expect(next.error).toBe("history request failed: 500");
  });
});
