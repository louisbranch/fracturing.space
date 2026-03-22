import type { HUDConnectionState } from "../interaction/player-hud/shared/contract";
import type { WireAIDebugTurnUpdate, WireChatMessage, WireRoomSnapshot } from "./types";

export type WSEvent =
  | { type: "ready"; snapshot: WireRoomSnapshot }
  | { type: "interaction.updated"; snapshot: WireRoomSnapshot }
  | { type: "chat.message"; message: WireChatMessage }
  | { type: "ai-debug.turn.updated"; update: WireAIDebugTurnUpdate }
  | { type: "typing"; participantId: string; name: string; active: boolean }
  | { type: "connection"; state: HUDConnectionState }
  | { type: "resync" }
  | { type: "error"; code: string; message: string };

type WSFrame = {
  type: string;
  request_id?: string;
  payload?: unknown;
};

type WSOptions = {
  campaignId: string;
  lastGameSeq: number;
  lastChatSeq: number;
  realtimeURL: string;
  onEvent: (event: WSEvent) => void;
};

export type WSConnection = {
  send: (frame: WSFrame) => void;
  close: () => void;
};

/** Wire frame types for the play WebSocket protocol. */
export const FrameType = {
  Connect: "play.connect",
  Ready: "play.ready",
  Ping: "play.ping",
  Pong: "play.pong",
  ChatMessage: "play.chat.message",
  AIDebugTurnUpdated: "play.ai_debug.turn.updated",
  ChatSend: "play.chat.send",
  Typing: "play.typing",
  InteractionUpdated: "play.interaction.updated",
  Error: "play.error",
  Resync: "play.resync",
} as const;

const PING_INTERVAL = 30_000;
const RECONNECT_BASE = 1_000;
const RECONNECT_MAX = 30_000;

export function connectWebSocket(opts: WSOptions): WSConnection {
  let ws: WebSocket | null = null;
  let pingTimer: ReturnType<typeof setInterval> | null = null;
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  let reconnectDelay = RECONNECT_BASE;
  let closed = false;
  let connected = false;

  function buildURL(): string {
    const loc = window.location;
    const protocol = loc.protocol === "https:" ? "wss:" : "ws:";
    return `${protocol}//${loc.host}${opts.realtimeURL}`;
  }

  // Track the latest known sequence so reconnects don't replay already-received
  // messages. Updated on every chat message and ready snapshot.
  let currentGameSeq = opts.lastGameSeq;
  let currentChatSeq = opts.lastChatSeq;

  function connect() {
    if (closed) return;
    ws = new WebSocket(buildURL());

    ws.onopen = () => {
      sendFrame({
        type: FrameType.Connect,
        payload: {
          campaign_id: opts.campaignId,
          last_game_seq: currentGameSeq,
          last_chat_seq: currentChatSeq,
        },
      });
      pingTimer = setInterval(() => {
        sendFrame({ type: FrameType.Ping });
      }, PING_INTERVAL);
    };

    ws.onmessage = (event) => {
      let frame: WSFrame;
      try {
        frame = JSON.parse(event.data as string) as WSFrame;
      } catch (e) {
        if (import.meta.env.DEV) console.warn("play: unparseable ws frame", e);
        return;
      }
      handleFrame(frame);
    };

    ws.onclose = () => {
      cleanup();
      if (!closed) {
        if (connected) {
          opts.onEvent({ type: "connection", state: "reconnecting" });
        }
        connected = false;
        scheduleReconnect();
      }
    };

    ws.onerror = () => {
      // onclose fires after onerror, reconnection happens there.
    };
  }

  function handleFrame(frame: WSFrame) {
    const payload = frame.payload as Record<string, unknown> | undefined;
    switch (frame.type) {
      case FrameType.Ready: {
        connected = true;
        reconnectDelay = RECONNECT_BASE;
        const snapshot = payload as unknown as WireRoomSnapshot;
        if (typeof snapshot?.latest_game_sequence === "number") {
          currentGameSeq = Math.max(currentGameSeq, snapshot.latest_game_sequence);
        }
        if (snapshot?.chat?.latest_sequence_id) {
          currentChatSeq = snapshot.chat.latest_sequence_id;
        }
        console.info("[play websocket ready]", {
          campaignId: opts.campaignId,
          participants: snapshot?.participants?.length ?? 0,
          characterCatalogEntries: Object.keys(snapshot?.character_inspection_catalog ?? {}).length,
          latestGameSeq: snapshot?.latest_game_sequence ?? 0,
          latestChatSeq: snapshot?.chat?.latest_sequence_id ?? 0,
        });
        opts.onEvent({ type: "connection", state: "connected" });
        opts.onEvent({ type: "ready", snapshot });
        break;
      }
      case FrameType.InteractionUpdated: {
        const snapshot = payload as unknown as WireRoomSnapshot;
        if (typeof snapshot?.latest_game_sequence === "number") {
          currentGameSeq = Math.max(currentGameSeq, snapshot.latest_game_sequence);
        }
        if (typeof snapshot?.chat?.latest_sequence_id === "number") {
          currentChatSeq = Math.max(currentChatSeq, snapshot.chat.latest_sequence_id);
        }
        opts.onEvent({ type: "interaction.updated", snapshot });
        break;
      }
      case FrameType.ChatMessage: {
        const msg = (payload as { message?: WireChatMessage })?.message;
        if (msg) {
          if (msg.sequence_id > currentChatSeq) currentChatSeq = msg.sequence_id;
          opts.onEvent({ type: "chat.message", message: msg });
        }
        break;
      }
      case FrameType.AIDebugTurnUpdated: {
        const update = payload as WireAIDebugTurnUpdate | undefined;
        if (update?.turn?.id) {
          opts.onEvent({ type: "ai-debug.turn.updated", update });
        }
        break;
      }
      case FrameType.Typing: {
        const te = payload as { participant_id?: string; name?: string; active?: boolean } | undefined;
        if (te) {
          opts.onEvent({
            type: "typing",
            participantId: te.participant_id ?? "",
            name: te.name ?? "",
            active: te.active ?? false,
          });
        }
        break;
      }
      case FrameType.Resync:
        opts.onEvent({ type: "resync" });
        break;
      case FrameType.Pong:
        break;
      case FrameType.Error: {
        const err = payload as { code?: string; message?: string } | undefined;
        opts.onEvent({
          type: "error",
          code: (err?.code ?? "") as string,
          message: (err?.message ?? "") as string,
        });
        break;
      }
    }
  }

  function sendFrame(frame: WSFrame) {
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(frame));
    }
  }

  function cleanup() {
    if (pingTimer) {
      clearInterval(pingTimer);
      pingTimer = null;
    }
  }

  function scheduleReconnect() {
    if (closed) return;
    reconnectTimer = setTimeout(() => {
      reconnectDelay = Math.min(reconnectDelay * 2, RECONNECT_MAX);
      connect();
    }, reconnectDelay);
  }

  connect();

  return {
    send: sendFrame,
    close: () => {
      closed = true;
      cleanup();
      if (reconnectTimer) {
        clearTimeout(reconnectTimer);
        reconnectTimer = null;
      }
      ws?.close();
    },
  };
}
