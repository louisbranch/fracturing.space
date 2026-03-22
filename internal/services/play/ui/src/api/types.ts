// Server wire types matching the Go protocol package JSON output.

import type { DaggerheartCharacterCardData } from "../systems/daggerheart/character-card/contract";
import type { DaggerheartCharacterSheetData } from "../systems/daggerheart/character-sheet/contract";

export type BootstrapResponse = {
  campaign_id: string;
  ai_debug_enabled?: boolean;
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
  current_interaction?: WireGMInteraction;
  interaction_history?: WireGMInteraction[];
};

export type WireSceneCharacter = {
  character_id: string;
  name?: string;
  owner_participant_id?: string;
};

export type WireGMInteractionIllustration = {
  image_url?: string;
  alt?: string;
  caption?: string;
};

export type WireGMInteractionBeat = {
  beat_id: string;
  type?: string;
  text?: string;
};

export type WireGMInteraction = {
  interaction_id: string;
  scene_id?: string;
  phase_id?: string;
  participant_id?: string;
  title?: string;
  character_ids: string[];
  illustration?: WireGMInteractionIllustration;
  beats: WireGMInteractionBeat[];
  created_at?: string;
};

export type WirePlayerPhase = {
  phase_id: string;
  status?: string;
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

export type WireUsage = {
  input_tokens?: number;
  output_tokens?: number;
  reasoning_tokens?: number;
  total_tokens?: number;
};

export type WireAIDebugTurnSummary = {
  id: string;
  turn_token?: string;
  participant_id?: string;
  provider?: string;
  model?: string;
  status?: string;
  last_error?: string;
  usage?: WireUsage;
  started_at?: string;
  updated_at?: string;
  completed_at?: string;
  entry_count?: number;
};

export type WireAIDebugEntry = {
  sequence: number;
  kind?: string;
  tool_name?: string;
  payload?: string;
  payload_truncated?: boolean;
  call_id?: string;
  response_id?: string;
  is_error?: boolean;
  created_at?: string;
  usage?: WireUsage;
};

export type WireAIDebugTurn = WireAIDebugTurnSummary & {
  entries: WireAIDebugEntry[];
};

export type WireAIDebugTurnUpdate = {
  turn: WireAIDebugTurnSummary;
  appended_entries: WireAIDebugEntry[];
};

export type WireAIDebugTurnsPage = {
  turns: WireAIDebugTurnSummary[];
  next_page_token?: string;
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
  typing_ttl_ms?: number;
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
