import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { connectWebSocket, FrameType, type WSConnection, type WSEvent } from "./websocket";

// ---------------------------------------------------------------------------
// Mock WebSocket
// ---------------------------------------------------------------------------

class MockWebSocket {
  static instances: MockWebSocket[] = [];

  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onerror: (() => void) | null = null;
  readyState = 1; // OPEN
  sent: string[] = [];

  constructor(public url: string) {
    MockWebSocket.instances.push(this);
  }

  send(data: string) {
    this.sent.push(data);
  }

  close() {
    this.readyState = 3;
  }

  // --- test helpers ---

  simulateOpen() {
    this.onopen?.();
  }

  simulateClose() {
    this.onclose?.();
  }

  simulateMessage(frame: object) {
    this.onmessage?.({ data: JSON.stringify(frame) });
  }

  simulateRawMessage(raw: string) {
    this.onmessage?.({ data: raw });
  }

  sentFrames(): Array<{ type: string; payload?: unknown }> {
    return this.sent.map((s) => JSON.parse(s));
  }
}

// Attach the OPEN constant so sendFrame's readyState check works.
(MockWebSocket as unknown as { OPEN: number }).OPEN = 1;

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function defaultOpts(onEvent: (e: WSEvent) => void) {
  return {
    campaignId: "camp-1",
    lastGameSeq: 10,
    lastChatSeq: 5,
    realtimeURL: "/realtime",
    onEvent,
  };
}

/** Return the most recently created MockWebSocket instance. */
function latestWS(): MockWebSocket {
  return MockWebSocket.instances[MockWebSocket.instances.length - 1];
}

/** Build a minimal WireRoomSnapshot payload. */
function readySnapshot(chatSeq: number) {
  return {
    interaction_state: { campaign_id: "camp-1" },
    participants: [],
    character_inspection_catalog: null,
    chat: {
      session_id: "s1",
      latest_sequence_id: chatSeq,
      messages: [],
      history_url: "/api/campaigns/camp-1/chat/history",
    },
    latest_game_sequence: 10,
  };
}

// ---------------------------------------------------------------------------
// Test suite
// ---------------------------------------------------------------------------

describe("connectWebSocket", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    MockWebSocket.instances = [];
    // jsdom provides window.location with protocol "http:" and host "localhost"
    // which is sufficient for buildURL().
    (globalThis as unknown as { WebSocket: typeof WebSocket }).WebSocket = MockWebSocket as unknown as typeof WebSocket;
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  // -----------------------------------------------------------------------
  // 1. Connection and ready handshake
  // -----------------------------------------------------------------------

  it("sends play.connect on open and emits connection+ready on play.ready", () => {
    const events: WSEvent[] = [];
    connectWebSocket(defaultOpts((e) => events.push(e)));

    const ws = latestWS();
    ws.simulateOpen();

    // Should have sent exactly one frame: play.connect
    const frames = ws.sentFrames();
    expect(frames).toHaveLength(1);
    expect(frames[0].type).toBe(FrameType.Connect);
    expect(frames[0].payload).toEqual({
      campaign_id: "camp-1",
      last_game_seq: 10,
      last_chat_seq: 5,
    });

    // Simulate server responding with play.ready
    ws.simulateMessage({ type: FrameType.Ready, payload: readySnapshot(5) });

    expect(events).toHaveLength(2);
    expect(events[0]).toEqual({ type: "connection", state: "connected" });
    expect(events[1].type).toBe("ready");
  });

  it("emits interaction.updated when the server pushes a full snapshot update", () => {
    const events: WSEvent[] = [];
    connectWebSocket(defaultOpts((e) => events.push(e)));

    const ws = latestWS();
    ws.simulateOpen();
    ws.simulateMessage({ type: FrameType.InteractionUpdated, payload: readySnapshot(6) });

    expect(events).toHaveLength(1);
    expect(events[0].type).toBe("interaction.updated");
    if (events[0].type === "interaction.updated") {
      expect(events[0].snapshot.latest_game_sequence).toBe(10);
      expect(events[0].snapshot.chat.latest_sequence_id).toBe(6);
    }
  });

  // -----------------------------------------------------------------------
  // 2. Chat message handling
  // -----------------------------------------------------------------------

  it("emits chat.message events with sequence tracking", () => {
    const events: WSEvent[] = [];
    connectWebSocket(defaultOpts((e) => events.push(e)));

    const ws = latestWS();
    ws.simulateOpen();
    ws.simulateMessage({ type: FrameType.Ready, payload: readySnapshot(5) });
    events.length = 0; // clear handshake events

    const msg = {
      message_id: "m1",
      campaign_id: "camp-1",
      session_id: "s1",
      sequence_id: 6,
      sent_at: "2026-03-19T12:00:00Z",
      actor: { participant_id: "p1", name: "Avery" },
      body: "Hello!",
    };

    ws.simulateMessage({ type: FrameType.ChatMessage, payload: { message: msg } });

    expect(events).toHaveLength(1);
    expect(events[0].type).toBe("chat.message");
    if (events[0].type === "chat.message") {
      expect(events[0].message.sequence_id).toBe(6);
      expect(events[0].message.body).toBe("Hello!");
    }
  });

  // -----------------------------------------------------------------------
  // 3. Typing events
  // -----------------------------------------------------------------------

  it("emits typing events from play.typing frames", () => {
    const events: WSEvent[] = [];
    connectWebSocket(defaultOpts((e) => events.push(e)));

    const ws = latestWS();
    ws.simulateOpen();
    ws.simulateMessage({
      type: FrameType.Typing,
      payload: { participant_id: "p2", name: "GM Bran", active: true },
    });

    expect(events).toHaveLength(1);
    expect(events[0]).toEqual({
      type: "typing",
      participantId: "p2",
      name: "GM Bran",
      active: true,
    });
  });

  // -----------------------------------------------------------------------
  // 4. Ping keepalive
  // -----------------------------------------------------------------------

  it("sends ping frames on a 30s interval after open", () => {
    connectWebSocket(defaultOpts(() => {}));

    const ws = latestWS();
    ws.simulateOpen();
    ws.sent.length = 0; // clear the connect frame

    // Advance 30 seconds — first ping
    vi.advanceTimersByTime(30_000);
    expect(ws.sentFrames()).toHaveLength(1);
    expect(ws.sentFrames()[0].type).toBe(FrameType.Ping);

    // Advance another 30 seconds — second ping
    vi.advanceTimersByTime(30_000);
    expect(ws.sentFrames()).toHaveLength(2);
  });

  // -----------------------------------------------------------------------
  // 5. Reconnection on close
  // -----------------------------------------------------------------------

  it("schedules reconnection with backoff when ws closes", () => {
    const events: WSEvent[] = [];
    connectWebSocket(defaultOpts((e) => events.push(e)));

    const ws1 = latestWS();
    ws1.simulateOpen();
    ws1.simulateMessage({ type: FrameType.Ready, payload: readySnapshot(5) });
    events.length = 0;

    // Simulate close
    ws1.simulateClose();

    // Should emit reconnecting since we were previously connected.
    expect(events).toHaveLength(1);
    expect(events[0]).toEqual({ type: "connection", state: "reconnecting" });

    // No new WS yet — reconnect is scheduled with delay
    expect(MockWebSocket.instances).toHaveLength(1);

    // Advance past the 1s base reconnect delay
    vi.advanceTimersByTime(1_000);

    // A new WebSocket should now exist
    expect(MockWebSocket.instances).toHaveLength(2);
  });

  // -----------------------------------------------------------------------
  // 6. Backoff reset on play.ready
  // -----------------------------------------------------------------------

  it("resets backoff delay on play.ready, not on TCP open", () => {
    connectWebSocket(defaultOpts(() => {}));

    const ws1 = latestWS();
    ws1.simulateOpen();
    ws1.simulateMessage({ type: FrameType.Ready, payload: readySnapshot(5) });

    // Simulate close — scheduleReconnect called with reconnectDelay = 1000
    ws1.simulateClose();
    vi.advanceTimersByTime(1_000);

    // Second connection opens but does NOT receive play.ready before closing.
    // This means backoff should NOT reset — it should double.
    const ws2 = latestWS();
    ws2.simulateOpen();
    ws2.simulateClose();

    // After the doubled 2s delay, reconnect should happen.
    // At 1.5s, no new connection yet.
    vi.advanceTimersByTime(1_500);
    expect(MockWebSocket.instances).toHaveLength(2);

    // At 2s total, the reconnect fires.
    vi.advanceTimersByTime(500);
    expect(MockWebSocket.instances).toHaveLength(3);

    // Third connection gets play.ready — backoff should reset to 1000.
    const ws3 = latestWS();
    ws3.simulateOpen();
    ws3.simulateMessage({ type: FrameType.Ready, payload: readySnapshot(5) });

    // Close and verify reset backoff: should reconnect after 1s, not 4s.
    ws3.simulateClose();
    vi.advanceTimersByTime(1_000);
    expect(MockWebSocket.instances).toHaveLength(4);
  });

  // -----------------------------------------------------------------------
  // 7. Error handling
  // -----------------------------------------------------------------------

  it("emits error events from play.error frames", () => {
    const events: WSEvent[] = [];
    connectWebSocket(defaultOpts((e) => events.push(e)));

    const ws = latestWS();
    ws.simulateOpen();
    ws.simulateMessage({
      type: FrameType.Error,
      payload: { code: "CAMPAIGN_NOT_FOUND", message: "No such campaign" },
    });

    expect(events).toHaveLength(1);
    expect(events[0]).toEqual({
      type: "error",
      code: "CAMPAIGN_NOT_FOUND",
      message: "No such campaign",
    });
  });

  it("emits error with empty defaults when payload fields are missing", () => {
    const events: WSEvent[] = [];
    connectWebSocket(defaultOpts((e) => events.push(e)));

    const ws = latestWS();
    ws.simulateOpen();
    ws.simulateMessage({ type: FrameType.Error, payload: {} });

    expect(events).toHaveLength(1);
    expect(events[0]).toEqual({ type: "error", code: "", message: "" });
  });

  it("emits resync events from play.resync frames", () => {
    const events: WSEvent[] = [];
    connectWebSocket(defaultOpts((e) => events.push(e)));

    const ws = latestWS();
    ws.simulateOpen();
    ws.simulateMessage({ type: FrameType.Resync });

    expect(events).toEqual([{ type: "resync" }]);
  });

  // -----------------------------------------------------------------------
  // 8. Unparseable frame
  // -----------------------------------------------------------------------

  it("silently drops malformed JSON without crashing", () => {
    const events: WSEvent[] = [];
    connectWebSocket(defaultOpts((e) => events.push(e)));

    const ws = latestWS();
    ws.simulateOpen();

    // Should not throw
    expect(() => ws.simulateRawMessage("{invalid json!!!")).not.toThrow();
    expect(events).toHaveLength(0);
  });

  // -----------------------------------------------------------------------
  // 9. Close method
  // -----------------------------------------------------------------------

  it("calling close() stops reconnection", () => {
    const conn = connectWebSocket(defaultOpts(() => {}));

    const ws1 = latestWS();
    ws1.simulateOpen();
    ws1.simulateMessage({ type: FrameType.Ready, payload: readySnapshot(5) });

    // Close the connection explicitly
    conn.close();

    // Simulate the ws onclose callback that follows
    ws1.simulateClose();

    // Even after advancing well past any backoff, no reconnection should happen.
    vi.advanceTimersByTime(60_000);
    expect(MockWebSocket.instances).toHaveLength(1);
  });

  it("calling close() clears pending reconnect timer", () => {
    const conn = connectWebSocket(defaultOpts(() => {}));

    const ws1 = latestWS();
    ws1.simulateOpen();
    ws1.simulateMessage({ type: FrameType.Ready, payload: readySnapshot(5) });

    // Trigger close which schedules reconnect
    ws1.simulateClose();

    // Now call close() before the timer fires — should cancel it
    conn.close();

    vi.advanceTimersByTime(60_000);
    expect(MockWebSocket.instances).toHaveLength(1);
  });

  // -----------------------------------------------------------------------
  // 10. Sequence tracking
  // -----------------------------------------------------------------------

  it("uses updated currentChatSeq from play.ready snapshot on reconnect", () => {
    connectWebSocket(defaultOpts(() => {}));

    const ws1 = latestWS();
    ws1.simulateOpen();

    // Initial connect should use lastChatSeq = 5
    expect(ws1.sentFrames()[0].payload).toEqual(
      expect.objectContaining({ last_chat_seq: 5 }),
    );

    // play.ready delivers a snapshot with higher sequence
    ws1.simulateMessage({ type: FrameType.Ready, payload: readySnapshot(20) });

    // Close and reconnect
    ws1.simulateClose();
    vi.advanceTimersByTime(1_000);

    const ws2 = latestWS();
    ws2.simulateOpen();

    // Reconnect should use updated sequence 20
    expect(ws2.sentFrames()[0].payload).toEqual(
      expect.objectContaining({ last_chat_seq: 20 }),
    );
  });

  it("uses updated currentGameSeq from interaction snapshots on reconnect", () => {
    connectWebSocket(defaultOpts(() => {}));

    const ws1 = latestWS();
    ws1.simulateOpen();
    ws1.simulateMessage({
      type: FrameType.InteractionUpdated,
      payload: {
        ...readySnapshot(5),
        latest_game_sequence: 42,
      },
    });

    ws1.simulateClose();
    vi.advanceTimersByTime(1_000);

    const ws2 = latestWS();
    ws2.simulateOpen();

    expect(ws2.sentFrames()[0].payload).toEqual(
      expect.objectContaining({ last_game_seq: 42 }),
    );
  });

  it("updates currentChatSeq from chat messages for reconnect", () => {
    connectWebSocket(defaultOpts(() => {}));

    const ws1 = latestWS();
    ws1.simulateOpen();
    ws1.simulateMessage({ type: FrameType.Ready, payload: readySnapshot(5) });

    // Receive chat messages that advance the sequence
    ws1.simulateMessage({
      type: FrameType.ChatMessage,
      payload: {
        message: {
          message_id: "m1",
          campaign_id: "camp-1",
          session_id: "s1",
          sequence_id: 42,
          sent_at: "2026-03-19T12:00:00Z",
          actor: { participant_id: "p1", name: "Avery" },
          body: "Hello",
        },
      },
    });

    // Close and reconnect
    ws1.simulateClose();
    vi.advanceTimersByTime(1_000);

    const ws2 = latestWS();
    ws2.simulateOpen();

    // Should use the highest seen sequence (42), not the snapshot value (5)
    expect(ws2.sentFrames()[0].payload).toEqual(
      expect.objectContaining({ last_chat_seq: 42 }),
    );
  });

  it("does not regress currentChatSeq from a lower-sequence chat message", () => {
    connectWebSocket(defaultOpts(() => {}));

    const ws1 = latestWS();
    ws1.simulateOpen();
    ws1.simulateMessage({ type: FrameType.Ready, payload: readySnapshot(50) });

    // Receive a message with a lower sequence (replay scenario)
    ws1.simulateMessage({
      type: FrameType.ChatMessage,
      payload: {
        message: {
          message_id: "m-old",
          campaign_id: "camp-1",
          session_id: "s1",
          sequence_id: 30,
          sent_at: "2026-03-19T11:00:00Z",
          actor: { participant_id: "p1", name: "Avery" },
          body: "Old message",
        },
      },
    });

    // Close and reconnect
    ws1.simulateClose();
    vi.advanceTimersByTime(1_000);

    const ws2 = latestWS();
    ws2.simulateOpen();

    // Should still be 50 — the ready snapshot value — not regressed to 30
    expect(ws2.sentFrames()[0].payload).toEqual(
      expect.objectContaining({ last_chat_seq: 50 }),
    );
  });
});
