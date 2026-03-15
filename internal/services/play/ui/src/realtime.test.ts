import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  PlayRealtimeClient,
  type RealtimeStatus,
  type ServerFrame,
} from "./realtime";
import type { PlayBootstrap } from "./protocol";
import { MockWebSocket } from "./test/fakeWebSocket";

const bootstrap: PlayBootstrap = {
  campaign_id: "camp-1",
  viewer: { participant_id: "player-1", name: "Avery", role: "PLAYER" },
  system: { id: "daggerheart", version: "1.0.0", name: "Daggerheart" },
  interaction_state: {
    campaign_id: "camp-1",
    campaign_name: "Guildhouse",
    gm_authority_participant_id: "gm-1",
  },
  chat: {
    session_id: "sess-1",
    latest_sequence_id: 10,
    messages: [],
    history_url: "/api/campaigns/camp-1/chat/history",
  },
  realtime: { url: "/ws/play", protocol_version: 1 },
};

describe("PlayRealtimeClient", () => {
  beforeEach(() => {
    MockWebSocket.reset();
    vi.stubGlobal("WebSocket", MockWebSocket as unknown as typeof WebSocket);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
    MockWebSocket.reset();
  });

  it("sends the connect frame when the socket opens", () => {
    const statuses: RealtimeStatus[] = [];

    new PlayRealtimeClient(
      bootstrap,
      () => undefined,
      (status) => {
        statuses.push(status);
      },
      17,
      33,
    );

    const socket = MockWebSocket.instances[0];
    expect(socket).toBeDefined();

    socket?.emitOpen();

    expect(statuses).toEqual([{ type: "open" }]);
    expect(JSON.parse(socket?.sent[0] ?? "null")).toEqual({
      type: "play.connect",
      payload: {
        campaign_id: "camp-1",
        last_game_seq: 17,
        last_chat_seq: 33,
      },
    });
  });

  it("reports an unexpected disconnect only once even if error is followed by close", () => {
    const statuses: RealtimeStatus[] = [];

    new PlayRealtimeClient(
      bootstrap,
      () => undefined,
      (status) => {
        statuses.push(status);
      },
      0,
      10,
    );

    const socket = MockWebSocket.instances[0];
    socket?.emitError();
    socket?.emitClose({ code: 1006 });

    expect(statuses).toEqual([{ type: "disconnected", message: "realtime connection error" }]);
  });

  it("treats manual close as a local shutdown instead of a disconnect", () => {
    const frames: ServerFrame[] = [];
    const statuses: RealtimeStatus[] = [];

    const client = new PlayRealtimeClient(
      bootstrap,
      (frame) => {
        frames.push(frame);
      },
      (status) => {
        statuses.push(status);
      },
      0,
      10,
    );

    client.close();

    expect(frames).toEqual([]);
    expect(statuses).toEqual([{ type: "closed" }]);
  });
});
