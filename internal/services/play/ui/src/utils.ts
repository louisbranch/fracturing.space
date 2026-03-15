import type {
  ChatHistoryResponse,
  InteractionState,
  PlayBootstrap,
  PlayChatMessage,
  PlayRoomSnapshot,
} from "./protocol";

export function resolveCampaignId(pathname: string): string {
  const match = pathname.match(/^\/campaigns\/([^/?#]+)/);
  if (!match?.[1]) {
    throw new Error(`unsupported play route: ${pathname}`);
  }
  return decodeURIComponent(match[1]);
}

export function normalizeBootstrap(payload: PlayBootstrap): PlayBootstrap {
  return {
    ...payload,
    chat: {
      ...payload.chat,
      messages: payload.chat.messages ?? [],
    },
    interaction_state: normalizeInteractionState(payload.interaction_state),
  };
}

export function normalizeSnapshot(
  bootstrap: PlayBootstrap,
  snapshot?: PlayRoomSnapshot,
  currentChat?: PlayBootstrap["chat"],
): PlayRoomSnapshot {
  if (!snapshot) {
    return {
      interaction_state: bootstrap.interaction_state,
      latest_game_sequence: 0,
      chat: bootstrap.chat,
    };
  }
  return {
    interaction_state: normalizeInteractionState(snapshot.interaction_state),
    latest_game_sequence: snapshot.latest_game_sequence ?? 0,
    chat: {
      session_id: snapshot.chat?.session_id ?? currentChat?.session_id ?? bootstrap.chat.session_id,
      latest_sequence_id:
        snapshot.chat?.latest_sequence_id ?? currentChat?.latest_sequence_id ?? bootstrap.chat.latest_sequence_id,
      messages: snapshot.chat?.messages ?? currentChat?.messages ?? bootstrap.chat.messages ?? [],
      history_url: snapshot.chat?.history_url ?? currentChat?.history_url ?? bootstrap.chat.history_url,
    },
  };
}

export function normalizeHistory(response: ChatHistoryResponse): PlayChatMessage[] {
  return response.messages ?? [];
}

export function mergeMessages(
  current: PlayChatMessage[],
  incoming: PlayChatMessage[],
): PlayChatMessage[] {
  const byID = new Map<string, PlayChatMessage>();
  for (const message of current) {
    byID.set(message.message_id, message);
  }
  for (const message of incoming) {
    byID.set(message.message_id, message);
  }
  return Array.from(byID.values()).sort((left, right) => left.sequence_id - right.sequence_id);
}

export function realtimeURL(rawURL: string): string {
  if (rawURL.startsWith("ws://") || rawURL.startsWith("wss://")) {
    return rawURL;
  }
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const url = new URL(rawURL, window.location.origin);
  url.protocol = protocol;
  return url.toString();
}

export function formatClock(isoTime?: string): string {
  if (!isoTime) {
    return "";
  }
  const date = new Date(isoTime);
  return Number.isNaN(date.getTime())
    ? ""
    : date.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });
}

function normalizeInteractionState(state: InteractionState): InteractionState {
  return {
    ...state,
    active_scene: state.active_scene
      ? {
          ...state.active_scene,
          characters: state.active_scene.characters ?? [],
        }
      : undefined,
    player_phase: state.player_phase
      ? {
          ...state.player_phase,
          acting_character_ids: state.player_phase.acting_character_ids ?? [],
          acting_participant_ids: state.player_phase.acting_participant_ids ?? [],
          slots: (state.player_phase.slots ?? []).map((slot) => ({
            ...slot,
            character_ids: slot.character_ids ?? [],
            review_character_ids: slot.review_character_ids ?? [],
          })),
        }
      : undefined,
    ooc: state.ooc
      ? {
          ...state.ooc,
          posts: state.ooc.posts ?? [],
          ready_to_resume_participant_ids: state.ooc.ready_to_resume_participant_ids ?? [],
        }
      : { open: false, posts: [], ready_to_resume_participant_ids: [] },
  };
}
