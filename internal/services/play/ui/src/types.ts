import type { JSX } from "react";

export interface ViewerSummary {
  participant_id: string;
  name: string;
  role: string;
}

export interface SystemSummary {
  id: string;
  version: string;
  name: string;
}

export interface InteractionSession {
  session_id: string;
  name: string;
}

export interface InteractionCharacter {
  character_id: string;
  name: string;
  owner_participant_id: string;
}

export interface InteractionGMOutput {
  text: string;
  participant_id: string;
  updated_at?: string;
}

export interface InteractionScene {
  scene_id: string;
  session_id: string;
  name: string;
  description: string;
  characters: InteractionCharacter[];
  gm_output?: InteractionGMOutput;
}

export interface ScenePlayerSlot {
  participant_id: string;
  summary_text: string;
  character_ids: string[];
  yielded: boolean;
  review_status: number;
  review_reason: string;
  review_character_ids: string[];
  updated_at?: string;
}

export interface ScenePlayerPhase {
  phase_id: string;
  status: number;
  frame_text: string;
  acting_character_ids: string[];
  acting_participant_ids: string[];
  slots: ScenePlayerSlot[];
}

export interface OOCPost {
  post_id: string;
  participant_id: string;
  body: string;
  created_at?: string;
}

export interface OOCState {
  open: boolean;
  posts: OOCPost[];
  ready_to_resume_participant_ids: string[];
}

export interface AITurnState {
  status: number;
  turn_token: string;
  owner_participant_id: string;
  source_event_type: string;
  source_scene_id: string;
  source_phase_id: string;
  last_error: string;
}

export interface InteractionState {
  campaign_id: string;
  campaign_name: string;
  viewer?: ViewerSummary;
  active_session?: InteractionSession;
  active_scene?: InteractionScene;
  player_phase?: ScenePlayerPhase;
  ooc?: OOCState;
  gm_authority_participant_id: string;
  ai_turn?: AITurnState;
}

export interface ChatInfo {
  session_id: string;
  latest_sequence_id: number;
  history_url: string;
}

export interface RealtimeInfo {
  url: string;
  protocol_version: number;
}

export interface PlayBootstrap {
  campaign_id: string;
  viewer?: ViewerSummary;
  system: SystemSummary;
  interaction_state: InteractionState;
  chat: ChatInfo;
  realtime: RealtimeInfo;
}

export interface PlayRoomSnapshot {
  interaction_state: InteractionState;
  latest_game_sequence: number;
  chat?: ChatInfo;
}

export interface PlayMessageActor {
  participant_id: string;
  name: string;
}

export interface PlayChatMessage {
  message_id: string;
  campaign_id: string;
  session_id: string;
  sequence_id: number;
  sent_at: string;
  actor: PlayMessageActor;
  body: string;
  client_message_id?: string;
}

export interface TypingEvent {
  campaign_id?: string;
  session_id: string;
  participant_id: string;
  name: string;
  active: boolean;
}

export interface ChatHistoryResponse {
  messages: PlayChatMessage[];
}

export type SystemRendererProps = {
  bootstrap: PlayBootstrap;
  snapshot: PlayRoomSnapshot;
};

export interface SystemRenderer {
  id: string;
  render(props: SystemRendererProps): JSX.Element;
}
