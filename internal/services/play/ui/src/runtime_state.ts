import type {
  PlayBootstrap,
  PlayChatMessage,
  PlayRoomSnapshot,
  TypingEvent,
} from "./protocol";
import type { RealtimeStatus, ServerFrame } from "./realtime";
import {
  mergeMessages,
  normalizeHistory,
  normalizeSnapshot,
} from "./utils";

type TypingState = Record<string, TypingEvent>;

export type RuntimeBootstrapData = {
  bootstrap: PlayBootstrap;
  snapshot: PlayRoomSnapshot;
  messages: PlayChatMessage[];
};

export type RuntimeState = {
  bootstrap?: PlayBootstrap;
  snapshot?: PlayRoomSnapshot;
  messages: PlayChatMessage[];
  chatTyping: TypingState;
  draftTyping: TypingState;
  error: string;
  loading: boolean;
  connected: boolean;
  loadingHistory: boolean;
};

export const initialRuntimeState: RuntimeState = {
  messages: [],
  chatTyping: {},
  draftTyping: {},
  error: "",
  loading: true,
  connected: false,
  loadingHistory: false,
};

export function createLoadedRuntimeState(runtime: RuntimeBootstrapData): RuntimeState {
  return {
    ...initialRuntimeState,
    bootstrap: runtime.bootstrap,
    snapshot: runtime.snapshot,
    messages: runtime.messages,
    loading: false,
  };
}

export function createFailedRuntimeState(message: string): RuntimeState {
  return {
    ...initialRuntimeState,
    loading: false,
    error: message,
  };
}

export function applyHistoryLoadStarted(current: RuntimeState): RuntimeState {
  return {
    ...current,
    loadingHistory: true,
  };
}

export function applyHistoryLoaded(
  current: RuntimeState,
  payload: { messages: PlayChatMessage[] },
): RuntimeState {
  return {
    ...current,
    loadingHistory: false,
    messages: mergeMessages(current.messages, normalizeHistory({ session_id: "", messages: payload.messages })),
    error: "",
  };
}

export function applyHistoryFailure(current: RuntimeState, message: string): RuntimeState {
  return {
    ...current,
    loadingHistory: false,
    error: message,
  };
}

export function applyRealtimeFrame(current: RuntimeState, frame: ServerFrame): RuntimeState {
  switch (frame.type) {
    case "play.ready":
    case "play.interaction.updated":
      if (!current.bootstrap) {
        return current;
      }
      return {
        ...current,
        snapshot: normalizeSnapshot(current.bootstrap, frame.payload, current.snapshot?.chat),
        connected: true,
      };
    case "play.chat.message":
      return {
        ...current,
        messages: mergeMessages(current.messages, [frame.payload.message]),
      };
    case "play.chat.typing":
      return {
        ...current,
        chatTyping: updateTypingState(current.chatTyping, frame.payload),
      };
    case "play.draft.typing":
      return {
        ...current,
        draftTyping: updateTypingState(current.draftTyping, frame.payload),
      };
    case "play.error":
      return {
        ...current,
        error: frame.payload.error.message,
      };
    case "play.resync":
      return {
        ...current,
        error: frame.payload.reason,
      };
    default:
      return current;
  }
}

export function applyRealtimeStatus(current: RuntimeState, status: RealtimeStatus): RuntimeState {
  switch (status.type) {
    case "open":
      return {
        ...current,
        connected: true,
      };
    case "closed":
      return {
        ...current,
        connected: false,
      };
    case "disconnected":
      return {
        ...current,
        connected: false,
        error: status.message,
      };
    default:
      return current;
  }
}

function updateTypingState(current: TypingState, event: TypingEvent): TypingState {
  const next = { ...current };
  if (!event.active) {
    delete next[event.participant_id];
    return next;
  }
  next[event.participant_id] = event;
  return next;
}
