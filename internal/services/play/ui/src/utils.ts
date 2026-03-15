import type {
  ChatHistoryResponse,
  InteractionState,
  InteractionSession,
  PlayBootstrap,
  PlayChatMessage,
  PlayRoomSnapshot,
} from "./types";

const SCENE_PHASE_GM = 1;
const SCENE_PHASE_PLAYERS = 2;
const SCENE_PHASE_GM_REVIEW = 3;

const AI_TURN_IDLE = 1;
const AI_TURN_QUEUED = 2;
const AI_TURN_RUNNING = 3;
const AI_TURN_FAILED = 4;

export function resolveCampaignId(pathname: string): string {
  const match = pathname.match(/^\/campaigns\/([^/?#]+)/);
  if (!match?.[1]) {
    throw new Error(`unsupported play route: ${pathname}`);
  }
  return decodeURIComponent(match[1]);
}

export function phaseLabel(status?: number): string {
  switch (status) {
    case SCENE_PHASE_PLAYERS:
      return "Players acting";
    case SCENE_PHASE_GM_REVIEW:
      return "GM reviewing";
    case SCENE_PHASE_GM:
      return "GM authority";
    default:
      return "Waiting for scene";
  }
}

export function aiTurnLabel(state: InteractionState): string {
  switch (state.ai_turn?.status) {
    case AI_TURN_QUEUED:
      return "Queued";
    case AI_TURN_RUNNING:
      return "Running";
    case AI_TURN_FAILED:
      return "Failed";
    case AI_TURN_IDLE:
      return "Idle";
    default:
      return "Unknown";
  }
}

export function oocLabel(state: InteractionState): string {
  const ooc = state.ooc;
  if (!ooc?.open) {
    return "In character";
  }
  if (ooc.ready_to_resume_participant_ids.length > 0) {
    return `OOC paused · ${ooc.ready_to_resume_participant_ids.length} ready`;
  }
  return "OOC paused";
}

export function normalizeBootstrap(payload: PlayBootstrap): PlayBootstrap {
  return {
    ...payload,
    interaction_state: {
      ...payload.interaction_state,
      active_scene: payload.interaction_state.active_scene
        ? {
            ...payload.interaction_state.active_scene,
            characters: payload.interaction_state.active_scene.characters ?? [],
          }
        : undefined,
      player_phase: payload.interaction_state.player_phase
        ? {
            ...payload.interaction_state.player_phase,
            acting_character_ids:
              payload.interaction_state.player_phase.acting_character_ids ?? [],
            acting_participant_ids:
              payload.interaction_state.player_phase.acting_participant_ids ?? [],
            slots: payload.interaction_state.player_phase.slots ?? [],
          }
        : undefined,
      ooc: payload.interaction_state.ooc
        ? {
            ...payload.interaction_state.ooc,
            posts: payload.interaction_state.ooc.posts ?? [],
            ready_to_resume_participant_ids:
              payload.interaction_state.ooc.ready_to_resume_participant_ids ?? [],
          }
        : { open: false, posts: [], ready_to_resume_participant_ids: [] },
    },
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
    interaction_state: snapshot.interaction_state,
    latest_game_sequence: snapshot.latest_game_sequence ?? 0,
    chat: {
      session_id: snapshot.chat?.session_id ?? currentChat?.session_id ?? bootstrap.chat.session_id,
      latest_sequence_id:
        snapshot.chat?.latest_sequence_id ?? currentChat?.latest_sequence_id ?? bootstrap.chat.latest_sequence_id,
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

export function sessionLabel(session?: InteractionSession): string {
  if (!session) {
    return "No active session";
  }
  if (session.name?.trim()) {
    return session.name;
  }
  return "Untitled session";
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
