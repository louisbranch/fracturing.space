import type {
  PlayBootstrap,
  PlayChatMessage,
  PlayRoomSnapshot,
  TypingEvent,
} from "./types";
import { realtimeURL } from "./utils";

type FrameHandler = (frame: ServerFrame) => void;
type OpenHandler = () => void;
type ErrorHandler = (message: string) => void;

type ClientFrame =
  | {
      type: "play.connect";
      payload: {
        campaign_id: string;
        last_game_seq: number;
        last_chat_seq: number;
      };
    }
  | {
      type: "play.chat.send";
      payload: {
        client_message_id: string;
        body: string;
      };
    }
  | {
      type: "play.chat.typing" | "play.draft.typing";
      payload: {
        active: boolean;
      };
    };

export type ServerFrame =
  | { type: "play.ready"; payload: PlayRoomSnapshot }
  | { type: "play.interaction.updated"; payload: PlayRoomSnapshot }
  | { type: "play.chat.message"; payload: { message: PlayChatMessage } }
  | { type: "play.chat.typing"; payload: TypingEvent }
  | { type: "play.draft.typing"; payload: TypingEvent }
  | { type: "play.error"; payload: { code: string; message: string } }
  | { type: "play.resync"; payload: { reason: string } };

export class PlayRealtimeClient {
  private readonly socket: WebSocket;

  constructor(
    bootstrap: PlayBootstrap,
    onFrame: FrameHandler,
    onOpen: OpenHandler,
    onError: ErrorHandler,
    lastGameSeq: number,
    lastChatSeq: number,
  ) {
    const socket = new WebSocket(realtimeURL(bootstrap.realtime.url));
    this.socket = socket;

    socket.addEventListener("open", () => {
      this.send({
        type: "play.connect",
        payload: {
          campaign_id: bootstrap.campaign_id,
          last_game_seq: lastGameSeq,
          last_chat_seq: lastChatSeq,
        },
      });
      onOpen();
    });

    socket.addEventListener("message", (event) => {
      try {
        const frame = JSON.parse(String(event.data)) as ServerFrame;
        onFrame(frame);
      } catch (error) {
        const message = error instanceof Error ? error.message : "invalid realtime payload";
        onError(message);
      }
    });

    socket.addEventListener("error", () => {
      onError("realtime connection error");
    });
  }

  close(): void {
    this.socket.close();
  }

  sendChat(body: string): void {
    this.send({
      type: "play.chat.send",
      payload: {
        client_message_id: `cli_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`,
        body,
      },
    });
  }

  sendChatTyping(active: boolean): void {
    this.send({ type: "play.chat.typing", payload: { active } });
  }

  sendDraftTyping(active: boolean): void {
    this.send({ type: "play.draft.typing", payload: { active } });
  }

  private send(frame: ClientFrame): void {
    this.socket.send(JSON.stringify(frame));
  }
}
