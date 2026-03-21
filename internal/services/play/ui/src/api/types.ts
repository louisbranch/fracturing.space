// Server wire types matching the Go protocol package JSON output.

import type { DaggerheartCharacterCardData } from "../systems/daggerheart/character-card/contract";
import type { DaggerheartCharacterSheetData } from "../systems/daggerheart/character-sheet/contract";

export type BootstrapResponse = {
  campaign_id: string;
  viewer?: WireViewer;
  system: WireSystem;
  interaction_state: WireInteractionState;
  participants: WireParticipant[];
  character_inspection_catalog: Record<string, WireCharacterInspection> | null;
  chat: WireChatSnapshot;
  realtime: WireRealtimeConfig;
};

export type WireViewer = {
  participant_id: string;
  name: string;
  role?: string;
};

export type WireSystem = {
  id: string;
  version: string;
  name: string;
};

export type WireInteractionState = {
  campaign_id: string;
  campaign_name?: string;
  locale?: string;
  viewer?: WireViewer;
  active_session?: WireSession;
  active_scene?: WireScene;
  player_phase?: WirePlayerPhase;
  ooc?: WireOOCState;
  gm_authority_participant_id?: string;
  ai_turn?: WireAITurn;
};

export type WireSession = {
  session_id: string;
  name?: string;
};

export type WireScene = {
  scene_id: string;
  name?: string;
  description?: string;
  characters: WireSceneCharacter[];
  gm_output?: WireGMOutput;
};

export type WireSceneCharacter = {
  character_id: string;
  name?: string;
  owner_participant_id?: string;
};

export type WireGMOutput = {
  text?: string;
  participant_id?: string;
  updated_at?: string;
};

export type WirePlayerPhase = {
  phase_id: string;
  status?: string;
  frame_text?: string;
  acting_character_ids: string[];
  acting_participant_ids: string[];
  slots: WirePlayerSlot[];
};

export type WirePlayerSlot = {
  participant_id: string;
  summary_text?: string;
  character_ids: string[];
  updated_at?: string;
  yielded: boolean;
  review_status?: string;
  review_reason?: string;
  review_character_ids: string[];
};

export type WireOOCState = {
  open: boolean;
  posts: WireOOCPost[];
  ready_to_resume_participant_ids: string[];
};

export type WireOOCPost = {
  post_id: string;
  participant_id: string;
  body: string;
  created_at?: string;
};

export type WireAITurn = {
  status?: string;
  owner_participant_id?: string;
  last_error?: string;
};

export type WireParticipant = {
  id: string;
  name: string;
  role?: string;
  avatar_url?: string;
  character_ids?: string[];
};

export type WireCharacterInspection = {
  system: "daggerheart";
  card: DaggerheartCharacterCardData;
  sheet: DaggerheartCharacterSheetData;
};

export type WireRealtimeConfig = {
  url: string;
  protocol_version: number;
};

export type WireChatSnapshot = {
  session_id: string;
  latest_sequence_id: number;
  messages: WireChatMessage[];
  history_url: string;
};

export type WireChatMessage = {
  message_id: string;
  campaign_id: string;
  session_id: string;
  sequence_id: number;
  sent_at: string;
  actor: { participant_id: string; name: string };
  body: string;
  client_message_id?: string;
};

export type WireRoomSnapshot = {
  interaction_state: WireInteractionState;
  participants: WireParticipant[];
  character_inspection_catalog: Record<string, WireCharacterInspection> | null;
  chat: WireChatSnapshot;
  latest_game_sequence: number;
};
