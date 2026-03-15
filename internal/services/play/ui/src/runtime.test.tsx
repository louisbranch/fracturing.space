import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { usePlayRuntime } from "./runtime";
import type { PlayBootstrap, PlayChatMessage } from "./protocol";
import { MockWebSocket } from "./test/fakeWebSocket";

const initialMessages: PlayChatMessage[] = [
  {
    message_id: "msg-9",
    campaign_id: "camp-1",
    session_id: "sess-1",
    sequence_id: 9,
    sent_at: "2026-03-14T12:00:00Z",
    actor: { participant_id: "player-1", name: "Avery" },
    body: "Ready at the gate.",
  },
  {
    message_id: "msg-10",
    campaign_id: "camp-1",
    session_id: "sess-1",
    sequence_id: 10,
    sent_at: "2026-03-14T12:01:00Z",
    actor: { participant_id: "gm-1", name: "GM" },
    body: "The scene opens.",
  },
];

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
    messages: initialMessages,
    history_url: "/api/campaigns/camp-1/chat/history",
  },
  realtime: { url: "/ws/play", protocol_version: 1 },
};

describe("usePlayRuntime", () => {
  const fetchMock = vi.fn<typeof fetch>();

  beforeEach(() => {
    window.history.pushState({}, "", "/campaigns/camp-1");
    MockWebSocket.reset();
    fetchMock.mockReset();
    vi.stubGlobal("fetch", fetchMock);
    vi.stubGlobal("WebSocket", MockWebSocket as unknown as typeof WebSocket);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
    MockWebSocket.reset();
    window.history.pushState({}, "", "/");
  });

  it("loads bootstrap chat without an extra history request", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse(bootstrap));

    const { result } = renderHook(() => usePlayRuntime());

    await waitFor(() => {
      expect(result.current.state.bootstrap).toBeDefined();
    });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(result.current.state.messages.map((message) => message.sequence_id)).toEqual([9, 10]);
  });

  it("clears loadingHistory when loading older messages fails", async () => {
    fetchMock
      .mockResolvedValueOnce(jsonResponse(bootstrap))
      .mockResolvedValueOnce(jsonResponse({}, 500));

    const { result } = renderHook(() => usePlayRuntime());

    await waitFor(() => {
      expect(result.current.state.bootstrap).toBeDefined();
    });

    await act(async () => {
      await result.current.loadOlderMessages();
    });

    await waitFor(() => {
      expect(result.current.state.loadingHistory).toBe(false);
    });

    expect(String(fetchMock.mock.calls[1]?.[0])).toContain("before_seq=9");
    expect(result.current.state.messages.map((message) => message.sequence_id)).toEqual([9, 10]);
    expect(result.current.state.error).toBe("history request failed: 500");
  });

  it("keeps the bootstrap state available when realtime setup fails", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse(bootstrap));
    vi.stubGlobal(
      "WebSocket",
      class {
        constructor() {
          throw new Error("socket unavailable");
        }
      } as unknown as typeof WebSocket,
    );

    const { result } = renderHook(() => usePlayRuntime());

    await waitFor(() => {
      expect(result.current.state.loading).toBe(false);
    });

    expect(result.current.state.bootstrap?.campaign_id).toBe("camp-1");
    expect(result.current.state.snapshot?.chat?.latest_sequence_id).toBe(10);
    expect(result.current.state.connected).toBe(false);
    expect(result.current.state.error).toBe("socket unavailable");
  });

  it("marks the runtime disconnected when the realtime socket closes", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse(bootstrap));

    const { result } = renderHook(() => usePlayRuntime());

    await waitFor(() => {
      expect(result.current.state.bootstrap).toBeDefined();
    });

    const socket = MockWebSocket.instances[0];
    expect(socket).toBeDefined();

    act(() => {
      socket?.emitOpen();
    });

    await waitFor(() => {
      expect(result.current.state.connected).toBe(true);
    });

    act(() => {
      socket?.emitClose({ code: 1012, reason: "server restart" });
    });

    await waitFor(() => {
      expect(result.current.state.connected).toBe(false);
    });

    expect(result.current.state.error).toBe("server restart");
  });
});

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  } as Response;
}
