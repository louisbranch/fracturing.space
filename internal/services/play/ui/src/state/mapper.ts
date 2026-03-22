// Maps server wire types to PlayerHUD component state.

import type { BackstageMessage, BackstageMode, BackstageParticipant, BackstageResumeState, BackstageState } from "../interaction/player-hud/backstage/shared/contract";
import type {
  OnStageAIStatus,
  OnStageCharacterSummary,
  OnStageGMBeatType,
  OnStageGMInteraction,
  OnStageMode,
  OnStageParticipant,
  OnStageParticipantRailStatus,
  OnStageScene,
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
  WireGMInteraction,
  WireOOCState,
  WireParticipant,
  WirePlayerPhase,
  WirePlayerSlot,
  WireRoomSnapshot,
  WireScene,
} from "../api/types";

// Safe enum validators — fall back to a default when the server sends an unknown value.
const KNOWN_AI_STATUSES = new Set<string>(["idle", "queued", "running", "failed"]);
const KNOWN_REVIEW_STATES = new Set<string>(["open", "under-review", "accepted", "changes-requested"]);
const KNOWN_ROLES = new Set<string>(["player", "gm"]);
const KNOWN_GM_BEAT_TYPES = new Set<string>(["fiction", "prompt", "resolution", "consequence", "guidance"]);

function safeAIStatus(raw: string | undefined): OnStageAIStatus {
  return KNOWN_AI_STATUSES.has(raw ?? "") ? (raw as OnStageAIStatus) : "idle";
}

function safeReviewState(raw: string | undefined): OnStageSlotReviewState {
  return KNOWN_REVIEW_STATES.has(raw ?? "") ? (raw as OnStageSlotReviewState) : "open";
}

function safeRole(raw: string | undefined): "player" | "gm" {
  return KNOWN_ROLES.has(raw ?? "") ? (raw as "player" | "gm") : "player";
}

function safeGMBeatType(raw: string | undefined): OnStageGMBeatType {
  return KNOWN_GM_BEAT_TYPES.has(raw ?? "") ? (raw as OnStageGMBeatType) : "fiction";
}

/** Returns whether the viewer has marked themselves "ready to resume" in the current OOC break. */
export function isViewerReadyToResume(
  bootstrap: BootstrapResponse,
  snapshot: WireRoomSnapshot | null,
): boolean {
  const ooc = snapshot?.interaction_state?.ooc ?? bootstrap.interaction_state?.ooc;
  const viewerPID = bootstrap.viewer?.participant_id ?? "";
  return ooc?.ready_to_resume_participant_ids?.includes(viewerPID) ?? false;
}

export function mapToPlayerHUDState(
  bootstrap: BootstrapResponse,
  snapshot: WireRoomSnapshot | null,
  connectionState: HUDConnectionState,
  activeTab: HUDNavbarTab,
  chatMessages: WireChatMessage[],
  typingParticipantIDs: string[],
  backURL: string,
): PlayerHUDState {
  const state = snapshot?.interaction_state ?? bootstrap.interaction_state;
  const participants = snapshot?.participants ?? bootstrap.participants ?? [];
  const catalog = snapshot?.character_inspection_catalog ?? bootstrap.character_inspection_catalog ?? {};
  const viewerPID = state.viewer?.participant_id ?? bootstrap.viewer?.participant_id ?? "";
  const viewerRole = safeRole(state.viewer?.role ?? bootstrap.viewer?.role);
  const returnURL = backURL.trim() || "/";
  const typingByParticipant = new Set(
    typingParticipantIDs
      .map((participantID) => participantID.trim())
      .filter((participantID) => participantID.length > 0),
  );

  const inspectionCatalog = mapCharacterInspectionCatalog(catalog);

  return {
    activeTab,
    connectionState,
    // Campaign navigation should preserve the durable participant-to-character
    // ownership from bootstrap even when realtime snapshots omit character IDs.
    campaignNavigation: mapCampaignNavigation(returnURL, bootstrap.participants ?? [], viewerPID, inspectionCatalog),
    onStage: mapOnStageState(state, participants, typingByParticipant, viewerPID, viewerRole, inspectionCatalog),
    backstage: mapBackstageState(state, participants, typingByParticipant, viewerPID, chatMessages, inspectionCatalog),
    sideChat: mapSideChatState(viewerPID, participants, typingByParticipant, chatMessages, inspectionCatalog),
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
    avatarUrl: catalog[id]?.card?.portrait?.src?.trim() || undefined,
  }));
}

function mapBaseParticipant(p: WireParticipant, catalog: PlayerHUDCharacterInspectionCatalog) {
  return {
    id: p.id,
    name: p.name,
    role: safeRole(p.role),
    avatarUrl: p.avatar_url?.trim() || undefined,
    characters: characterRefsFromIDs(p.character_ids ?? [], catalog),
  };
}

function mapOnStageState(
  state: WireInteractionState,
  participants: WireParticipant[],
  typingByParticipant: ReadonlySet<string>,
  viewerPID: string,
  viewerRole: string,
  catalog: PlayerHUDCharacterInspectionCatalog,
): OnStageState {
  const scene = state.active_scene;
  const phase = state.player_phase;
  const ooc = state.ooc;
  const aiTurn = state.ai_turn;

  const mode = deriveOnStageMode(phase, ooc, viewerPID);
  const aiStatus = safeAIStatus(aiTurn?.status);
  const currentInteraction = scene?.current_interaction;

  const onStageParticipants: OnStageParticipant[] = participants.map((p) =>
    mapOnStageParticipant(p, phase, typingByParticipant, state.gm_authority_participant_id, catalog),
  );

  const slots: OnStageSlot[] = (phase?.slots ?? []).map((s) => mapOnStageSlot(s, catalog));

  const viewerSlot = phase?.slots?.find((s) => s.participant_id === viewerPID);
  const viewerControls = deriveViewerControls(mode, viewerSlot, viewerRole);

  return {
    mode,
    aiStatus,
    aiOwnerParticipantId: aiTurn?.owner_participant_id?.trim() || undefined,
    scene: mapOnStageScene(scene, catalog),
    currentInteraction: mapGMInteraction(currentInteraction),
    interactionHistory: (scene?.interaction_history ?? [])
      .map(mapGMInteraction)
      .filter((interaction): interaction is OnStageGMInteraction => Boolean(interaction)),
    oocReason: undefined,
    viewerParticipantId: viewerPID,
    actingParticipantIds: phase?.acting_participant_ids ?? [],
    gmAuthorityParticipantId: state.gm_authority_participant_id,
    participants: onStageParticipants,
    slots,
    characterInspectionCatalog: catalog,
    viewerControls,
  };
}

function mapOnStageScene(
  scene: WireScene | undefined,
  catalog: PlayerHUDCharacterInspectionCatalog,
): OnStageScene {
  return {
    id: scene?.scene_id ?? "",
    name: scene?.name ?? "",
    description: scene?.description,
    characters: mapSceneCharacters(scene, catalog),
    resolvedInteractionCount: scene?.interaction_history?.length ?? 0,
  };
}

function mapSceneCharacters(
  scene: WireScene | undefined,
  catalog: PlayerHUDCharacterInspectionCatalog,
): OnStageCharacterSummary[] {
  return (scene?.characters ?? []).map((character) => ({
    id: character.character_id,
    name: catalog[character.character_id]?.card?.name ?? character.name ?? character.character_id,
    avatarUrl: catalog[character.character_id]?.card?.portrait?.src?.trim() || undefined,
  }));
}

function mapGMInteraction(interaction: WireGMInteraction | undefined): OnStageGMInteraction | undefined {
  if (!interaction) {
    return undefined;
  }

  return {
    id: interaction.interaction_id,
    title: interaction.title?.trim() || "GM Interaction",
    characterIds: interaction.character_ids ?? [],
    illustration: interaction.illustration?.image_url
      ? {
          imageUrl: interaction.illustration.image_url,
          alt: interaction.illustration.alt?.trim() || "GM interaction illustration",
          caption: interaction.illustration.caption?.trim() || undefined,
        }
      : undefined,
    beats: (interaction.beats ?? [])
      .map((beat) => {
        const text = beat.text?.trim() ?? "";
        if (!text) {
          return null;
        }
        return {
          id: beat.beat_id,
          type: safeGMBeatType(beat.type),
          text,
        };
      })
      .filter((beat): beat is NonNullable<typeof beat> => Boolean(beat)),
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
  typingByParticipant: ReadonlySet<string>,
  gmAuthorityPID: string | undefined,
  catalog: PlayerHUDCharacterInspectionCatalog,
): OnStageParticipant {
  let railStatus: OnStageParticipantRailStatus = "waiting";
  if (typingByParticipant.has(p.id)) {
    railStatus = "typing";
  } else {
    const slot = phase?.slots?.find((s) => s.participant_id === p.id);
    if (slot) {
      if (slot.review_status === "changes_requested") railStatus = "changes-requested";
      else if (slot.yielded) railStatus = "yielded";
      else if (phase?.acting_participant_ids?.includes(p.id)) railStatus = "active";
    }
  }

  return {
    ...mapBaseParticipant(p, catalog),
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
    reviewState: safeReviewState(s.review_status),
    reviewReason: s.review_reason,
  };
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
  typingByParticipant: ReadonlySet<string>,
  viewerPID: string,
  chatMessages: WireChatMessage[],
  catalog: PlayerHUDCharacterInspectionCatalog,
): BackstageState {
  const ooc = state.ooc;
  const mode: BackstageMode = ooc?.open ? "open" : "dormant";
  const resumeState = deriveBackstageResumeState(ooc, participants, viewerPID);

  const backstageParticipants: BackstageParticipant[] = participants.map((p) => ({
    ...mapBaseParticipant(p, catalog),
    readyToResume: ooc?.ready_to_resume_participant_ids?.includes(p.id) ?? false,
    typing: typingByParticipant.has(p.id),
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
  typingByParticipant: ReadonlySet<string>,
  chatMessages: WireChatMessage[],
  catalog: PlayerHUDCharacterInspectionCatalog,
): SideChatState {
  const sideChatParticipants: SideChatParticipant[] = participants.map((p) => ({
    ...mapBaseParticipant(p, catalog),
    typing: typingByParticipant.has(p.id),
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
