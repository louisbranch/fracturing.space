// Maps server wire types to PlayerHUD component state.

import type { BackstageMessage, BackstageMode, BackstageParticipant, BackstageResumeState, BackstageState } from "../interaction/player-hud/backstage/shared/contract";
import type {
  OnStageAIStatus,
  OnStageCharacterSummary,
  OnStageMode,
  OnStageParticipant,
  OnStageParticipantRailStatus,
  OnStageSlot,
  OnStageSlotReviewState,
  OnStageState,
  OnStageViewerControls,
} from "../interaction/player-hud/on-stage/shared/contract";
import type {
  PlayerHUDCharacterInspectionCatalog,
  PlayerHUDCharacterReference,
} from "../interaction/player-hud/shared/character-inspection-contract";
import type {
  HUDConnectionState,
  HUDNavbarTab,
  PlayerHUDCampaignNavigation,
  PlayerHUDCharacterController,
  PlayerHUDState,
  SideChatMessage,
  SideChatParticipant,
  SideChatState,
} from "../interaction/player-hud/shared/contract";
import type {
  BootstrapResponse,
  WireCharacterInspection,
  WireChatMessage,
  WireInteractionState,
  WireOOCState,
  WireParticipant,
  WirePlayerPhase,
  WirePlayerSlot,
  WireRoomSnapshot,
  WireScene,
} from "../api/types";

export function mapToPlayerHUDState(
  bootstrap: BootstrapResponse,
  snapshot: WireRoomSnapshot | null,
  connectionState: HUDConnectionState,
  activeTab: HUDNavbarTab,
  chatMessages: WireChatMessage[],
): PlayerHUDState {
  const state = snapshot?.interaction_state ?? bootstrap.interaction_state;
  const participants = snapshot?.participants ?? bootstrap.participants ?? [];
  const catalog = snapshot?.character_inspection_catalog ?? bootstrap.character_inspection_catalog ?? {};
  const viewerPID = state.viewer?.participant_id ?? bootstrap.viewer?.participant_id ?? "";
  const viewerRole = state.viewer?.role ?? bootstrap.viewer?.role ?? "player";
  const backURL = bootstrap.campaign_id
    ? `/app/campaigns/${bootstrap.campaign_id}/game`
    : "/";

  const inspectionCatalog = mapCharacterInspectionCatalog(catalog);

  return {
    activeTab,
    connectionState,
    campaignNavigation: mapCampaignNavigation(backURL, participants, viewerPID, inspectionCatalog),
    onStage: mapOnStageState(state, participants, viewerPID, viewerRole, inspectionCatalog),
    backstage: mapBackstageState(state, participants, viewerPID, chatMessages, inspectionCatalog),
    sideChat: mapSideChatState(viewerPID, participants, chatMessages, inspectionCatalog),
  };
}

function mapCharacterInspectionCatalog(
  wire: Record<string, WireCharacterInspection> | null,
): PlayerHUDCharacterInspectionCatalog {
  if (!wire) return {};
  const result: PlayerHUDCharacterInspectionCatalog = {};
  for (const [id, entry] of Object.entries(wire)) {
    result[id] = {
      system: entry.system,
      card: entry.card,
      sheet: entry.sheet,
    };
  }
  return result;
}

function mapCampaignNavigation(
  backURL: string,
  participants: WireParticipant[],
  viewerPID: string,
  catalog: PlayerHUDCharacterInspectionCatalog,
): PlayerHUDCampaignNavigation {
  const controllers: PlayerHUDCharacterController[] = participants.map((p) => ({
    participantId: p.id,
    participantName: p.name,
    isViewer: p.id === viewerPID,
    characters: characterRefsFromIDs(p.character_ids ?? [], catalog),
  }));
  return {
    returnHref: backURL,
    characterControllers: controllers,
    characterInspectionCatalog: catalog,
  };
}

function characterRefsFromIDs(
  ids: string[],
  catalog: PlayerHUDCharacterInspectionCatalog,
): PlayerHUDCharacterReference[] {
  return ids.map((id) => ({
    id,
    name: catalog[id]?.card?.name ?? id,
    avatarUrl: undefined,
  }));
}

function mapOnStageState(
  state: WireInteractionState,
  participants: WireParticipant[],
  viewerPID: string,
  viewerRole: string,
  catalog: PlayerHUDCharacterInspectionCatalog,
): OnStageState {
  const scene = state.active_scene;
  const phase = state.player_phase;
  const ooc = state.ooc;
  const aiTurn = state.ai_turn;

  const mode = deriveOnStageMode(phase, ooc, viewerPID);
  const aiStatus: OnStageAIStatus = (aiTurn?.status as OnStageAIStatus) ?? "idle";

  const onStageParticipants: OnStageParticipant[] = participants.map((p) =>
    mapOnStageParticipant(p, phase, state.gm_authority_participant_id, catalog),
  );

  const slots: OnStageSlot[] = (phase?.slots ?? []).map((s) => mapOnStageSlot(s, catalog));

  const viewerSlot = phase?.slots?.find((s) => s.participant_id === viewerPID);
  const viewerControls = deriveViewerControls(mode, viewerSlot, viewerRole);

  return {
    mode,
    aiStatus,
    sceneName: scene?.name ?? "",
    sceneDescription: scene?.description,
    gmOutputText: scene?.gm_output?.text,
    frameText: phase?.frame_text,
    oocReason: undefined,
    viewerParticipantId: viewerPID,
    actingParticipantIds: phase?.acting_participant_ids ?? [],
    actingCharacterNames: actingCharacterNames(phase, scene),
    gmAuthorityParticipantId: state.gm_authority_participant_id,
    participants: onStageParticipants,
    slots,
    characterInspectionCatalog: catalog,
    viewerControls,
  };
}

export function deriveOnStageMode(
  phase: WirePlayerPhase | undefined,
  ooc: WireOOCState | undefined,
  viewerPID: string,
): OnStageMode {
  if (ooc?.open) return "ooc-blocked";
  if (!phase || phase.status === "gm" || phase.status === "gm_review") return "waiting-on-gm";

  const viewerSlot = phase.slots?.find((s) => s.participant_id === viewerPID);
  if (viewerSlot?.review_status === "changes_requested") return "changes-requested";
  if (viewerSlot?.yielded) return "yielded-waiting";

  if (phase.status === "players" && phase.acting_participant_ids?.includes(viewerPID)) {
    return "acting";
  }

  return "waiting-on-gm";
}

function mapOnStageParticipant(
  p: WireParticipant,
  phase: WirePlayerPhase | undefined,
  gmAuthorityPID: string | undefined,
  catalog: PlayerHUDCharacterInspectionCatalog,
): OnStageParticipant {
  const slot = phase?.slots?.find((s) => s.participant_id === p.id);
  let railStatus: OnStageParticipantRailStatus = "waiting";
  if (slot) {
    if (slot.review_status === "changes_requested") railStatus = "changes-requested";
    else if (slot.yielded) railStatus = "yielded";
    else if (phase?.acting_participant_ids?.includes(p.id)) railStatus = "active";
  }

  return {
    id: p.id,
    name: p.name,
    role: (p.role as "player" | "gm") ?? "player",
    characters: characterRefsFromIDs(p.character_ids ?? [], catalog),
    railStatus,
    ownsGMAuthority: p.id === gmAuthorityPID ? true : undefined,
  };
}

function mapOnStageSlot(s: WirePlayerSlot, catalog: PlayerHUDCharacterInspectionCatalog): OnStageSlot {
  return {
    id: s.participant_id,
    participantId: s.participant_id,
    characters: characterRefsFromIDs(s.character_ids ?? [], catalog),
    body: s.summary_text,
    updatedAt: s.updated_at,
    yielded: s.yielded,
    reviewState: (s.review_status as OnStageSlotReviewState) ?? "open",
    reviewReason: s.review_reason,
  };
}

function actingCharacterNames(phase: WirePlayerPhase | undefined, scene: WireScene | undefined): string[] {
  const ids = phase?.acting_character_ids ?? [];
  if (!scene?.characters) return [];
  return ids
    .map((id) => scene.characters.find((c) => c.character_id === id)?.name ?? "")
    .filter(Boolean);
}

function deriveViewerControls(mode: OnStageMode, viewerSlot: WirePlayerSlot | undefined, viewerRole: string): OnStageViewerControls {
  const isActing = mode === "acting" || mode === "changes-requested";
  const isYielded = mode === "yielded-waiting";
  const hasSubmission = !!viewerSlot?.summary_text;

  return {
    canSubmit: isActing,
    canSubmitAndYield: isActing && !isYielded,
    canYield: isActing && !isYielded && hasSubmission,
    canUnyield: isYielded,
    disabledReason: mode === "ooc-blocked"
      ? "Session paused for OOC discussion"
      : mode === "waiting-on-gm" && viewerRole === "player"
        ? "Waiting for GM"
        : undefined,
  };
}

function mapBackstageState(
  state: WireInteractionState,
  participants: WireParticipant[],
  viewerPID: string,
  chatMessages: WireChatMessage[],
  catalog: PlayerHUDCharacterInspectionCatalog,
): BackstageState {
  const ooc = state.ooc;
  const mode: BackstageMode = ooc?.open ? "open" : "dormant";
  const resumeState = deriveBackstageResumeState(ooc, participants, viewerPID);

  const backstageParticipants: BackstageParticipant[] = participants.map((p) => ({
    id: p.id,
    name: p.name,
    role: (p.role as "player" | "gm") ?? "player",
    characters: characterRefsFromIDs(p.character_ids ?? [], catalog),
    readyToResume: ooc?.ready_to_resume_participant_ids?.includes(p.id) ?? false,
  }));

  const messages: BackstageMessage[] = (ooc?.posts ?? []).map((post) => ({
    id: post.post_id,
    participantId: post.participant_id,
    body: post.body,
    sentAt: post.created_at ?? "",
  }));

  return {
    mode,
    sceneName: state.active_scene?.name,
    gmAuthorityParticipantId: state.gm_authority_participant_id,
    resumeState,
    viewerParticipantId: viewerPID,
    participants: backstageParticipants,
    messages,
    characterInspectionCatalog: catalog,
  };
}

function deriveBackstageResumeState(
  ooc: WireOOCState | undefined,
  participants: WireParticipant[],
  viewerPID: string,
): BackstageResumeState {
  if (!ooc?.open) return "inactive";
  const readyIDs = ooc.ready_to_resume_participant_ids ?? [];
  const viewerRole = participants.find((p) => p.id === viewerPID)?.role;
  if (viewerRole === "gm" && readyIDs.length > 0) return "waiting-on-gm";
  if (readyIDs.length > 0 && readyIDs.length < participants.length) return "collecting-ready";
  if (readyIDs.length >= participants.length) return "waiting-on-gm";
  return "collecting-ready";
}

function mapSideChatState(
  viewerPID: string,
  participants: WireParticipant[],
  chatMessages: WireChatMessage[],
  catalog: PlayerHUDCharacterInspectionCatalog,
): SideChatState {
  const sideChatParticipants: SideChatParticipant[] = participants.map((p) => ({
    id: p.id,
    name: p.name,
    role: (p.role as "player" | "gm") ?? "player",
    characters: characterRefsFromIDs(p.character_ids ?? [], catalog),
  }));

  const messages: SideChatMessage[] = chatMessages.map((m) => ({
    id: m.message_id,
    participantId: m.actor.participant_id,
    body: m.body,
    sentAt: m.sent_at,
  }));

  return {
    viewerParticipantId: viewerPID,
    participants: sideChatParticipants,
    messages,
    characterInspectionCatalog: catalog,
  };
}
