import type {
  InteractionState,
  InteractionSession,
  PlayBootstrap,
  PlayRoomSnapshot,
  ScenePlayerSlot,
} from "./protocol";

const SCENE_PHASE_GM = 1;
const SCENE_PHASE_PLAYERS = 2;
const SCENE_PHASE_GM_REVIEW = 3;

const AI_TURN_IDLE = 1;
const AI_TURN_QUEUED = 2;
const AI_TURN_RUNNING = 3;
const AI_TURN_FAILED = 4;

export type PlayShellViewModel = {
  campaignName: string;
  viewerName: string;
  sessionLabel: string;
  systemLabel: string;
  connectedLabel: string;
};

export type SystemSceneViewModel = {
  title: string;
  description: string;
  characterCount: number;
  characters: Array<{ id: string; name: string }>;
};

export type SystemSlotViewModel = {
  key: string;
  participantLabel: string;
  summaryText: string;
  statusLabel: string;
};

export type SystemRenderViewModel = {
  campaignName: string;
  viewerName: string;
  sessionLabel: string;
  gmAuthorityLabel: string;
  scenePhaseLabel: string;
  oocLabel: string;
  aiTurnLabel: string;
  scene: SystemSceneViewModel;
  slots: SystemSlotViewModel[];
};

export function createPlayShellViewModel(
  bootstrap: PlayBootstrap,
  snapshot: PlayRoomSnapshot,
  connected: boolean,
): PlayShellViewModel {
  return {
    campaignName: snapshot.interaction_state.campaign_name || "Untitled campaign",
    viewerName: snapshot.interaction_state.viewer?.name || "participant",
    sessionLabel: formatSessionLabel(snapshot.interaction_state.active_session),
    systemLabel: bootstrap.system.name || bootstrap.system.id || "system",
    connectedLabel: connected ? "Connected" : "Disconnected",
  };
}

export function createSystemRenderViewModel(snapshot: PlayRoomSnapshot): SystemRenderViewModel {
  const state = snapshot.interaction_state;
  const scene = state.active_scene;
  const phase = state.player_phase;

  return {
    campaignName: state.campaign_name || "Untitled campaign",
    viewerName: state.viewer?.name || "Unknown participant",
    sessionLabel: formatSessionLabel(state.active_session),
    gmAuthorityLabel: state.gm_authority_participant_id || "Unassigned",
    scenePhaseLabel: scenePhaseLabel(phase?.status),
    oocLabel: formatOOCLabel(state),
    aiTurnLabel: formatAITurnLabel(state),
    scene: {
      title: scene?.name || "No active scene",
      description: scene?.description || "The GM has not opened a scene yet.",
      characterCount: scene?.characters?.length ?? 0,
      characters: (scene?.characters ?? []).map((character) => ({
        id: character.character_id,
        name: character.name || character.character_id,
      })),
    },
    slots: (phase?.slots ?? []).map((slot) => ({
      key: `${slot.participant_id}-${slot.updated_at || "slot"}`,
      participantLabel: slot.participant_id || "participant",
      summaryText: slot.summary_text || "Waiting for action.",
      statusLabel: slotStatusLabel(slot),
    })),
  };
}

export function formatSessionLabel(session?: InteractionSession): string {
  if (!session) {
    return "No active session";
  }
  if (session.name?.trim()) {
    return session.name;
  }
  return "Untitled session";
}

function scenePhaseLabel(status?: number): string {
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

function formatAITurnLabel(state: InteractionState): string {
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

function formatOOCLabel(state: InteractionState): string {
  const ooc = state.ooc;
  if (!ooc?.open) {
    return "In character";
  }
  if (ooc.ready_to_resume_participant_ids.length > 0) {
    return `OOC paused · ${ooc.ready_to_resume_participant_ids.length} ready`;
  }
  return "OOC paused";
}

function slotStatusLabel(slot: ScenePlayerSlot): string {
  return slot.updated_at ? "Updated" : "Pending";
}
