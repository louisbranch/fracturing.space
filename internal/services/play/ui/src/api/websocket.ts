import type { HUDConnectionState } from "../interaction/player-hud/shared/contract";
import type { WireChatMessage, WireRoomSnapshot } from "./types";

export type WSEvent =
  | { type: "ready"; snapshot: WireRoomSnapshot }
  | { type: "chat.message"; message: WireChatMessage }
  | { type: "chat.typing"; participantId: string; name: string; active: boolean }
  | { type: "draft.typing"; participantId: string; name: string; active: boolean }
  | { type: "connection"; state: HUDConnectionState }
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

  function connect() {
    if (closed) return;
    ws = new WebSocket(buildURL());

    ws.onopen = () => {
      reconnectDelay = RECONNECT_BASE;
      sendFrame({
        type: "play.connect",
        payload: {
          campaign_id: opts.campaignId,
          last_game_seq: opts.lastGameSeq,
          last_chat_seq: opts.lastChatSeq,
        },
      });
      pingTimer = setInterval(() => {
        sendFrame({ type: "play.ping" });
      }, PING_INTERVAL);
    };

    ws.onmessage = (event) => {
      let frame: WSFrame;
      try {
        frame = JSON.parse(event.data as string) as WSFrame;
      } catch {
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
      case "play.ready":
        connected = true;
        opts.onEvent({ type: "connection", state: "connected" });
        opts.onEvent({ type: "ready", snapshot: payload as unknown as WireRoomSnapshot });
        break;
      case "play.chat.message": {
        const msg = (payload as { message?: WireChatMessage })?.message;
        if (msg) opts.onEvent({ type: "chat.message", message: msg });
        break;
      }
      case "play.chat.typing": {
        const te = payload as { participant_id?: string; name?: string; active?: boolean } | undefined;
        if (te) {
          opts.onEvent({
            type: "chat.typing",
            participantId: te.participant_id ?? "",
            name: te.name ?? "",
            active: te.active ?? false,
          });
        }
        break;
      }
      case "play.draft.typing": {
        const te = payload as { participant_id?: string; name?: string; active?: boolean } | undefined;
        if (te) {
          opts.onEvent({
            type: "draft.typing",
            participantId: te.participant_id ?? "",
            name: te.name ?? "",
            active: te.active ?? false,
          });
        }
        break;
      }
      case "play.pong":
        break;
      case "play.error": {
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
