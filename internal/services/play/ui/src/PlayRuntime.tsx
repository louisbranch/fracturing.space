import { useCallback, useEffect, useReducer, useRef, useState } from "react";
import { fetchAIDebugTurn, fetchAIDebugTurns } from "./api/aiDebug";
import { fetchBootstrap } from "./api/bootstrap";
import * as mutations from "./api/mutations";
import type { WSConnection, WSEvent } from "./api/websocket";
import { FrameType, connectWebSocket } from "./api/websocket";
import type {
  BootstrapResponse,
  WireAIDebugTurn,
  WireAIDebugTurnSummary,
  WireAIDebugTurnUpdate,
  WireAIDebugTurnsPage,
  WireChatMessage,
  WireRoomSnapshot,
} from "./api/types";
import type { AIDebugPanelState } from "./interaction/player-hud/ai-debug/shared/contract";
import type { HUDConnectionState, HUDNavbarTab } from "./interaction/player-hud/shared/contract";
import { PlayerHUDShell } from "./interaction/player-hud/player-hud-shell/PlayerHUDShell";
import {
  PlayerHUDCharacterInspectorDialog,
  usePlayerHUDCharacterInspector,
} from "./interaction/player-hud/shared/PlayerHUDCharacterInspector";
import { isViewerReadyToResume, mapToPlayerHUDState } from "./state/mapper";
import type { PlayShellConfig } from "./shell_config";
import { TransitionPreferencesProvider } from "./transition/TransitionPreferencesContext";
import { useTransitionEffects } from "./transition/useTransitionEffects";
import { useTransitionAudio } from "./transition/useTransitionAudio";
import { SceneTransitionOverlay } from "./transition/SceneTransitionOverlay";
import { TransitionSettingsModal } from "./transition/TransitionSettingsModal";
import { useTransitionPreferences } from "./transition/TransitionPreferencesContext";

// --- State ---

type RuntimeState = {
  phase: "loading" | "ready" | "error";
  errorMessage?: string;
  mutationError?: string;
  bootstrap: BootstrapResponse | null;
  snapshot: WireRoomSnapshot | null;
  chatMessages: WireChatMessage[];
  remoteTypingParticipantIDs: string[];
  viewerTyping: boolean;
  connectionState: HUDConnectionState;
  activeTab: HUDNavbarTab;
  onStageDraft: string;
  backstageDraft: string;
  sideChatDraft: string;
  isSidebarOpen: boolean;
  aiDebug: AIDebugPanelState;
};

type RuntimeAction =
  | { type: "bootstrap-loaded"; bootstrap: BootstrapResponse }
  | { type: "runtime-resynced"; bootstrap: BootstrapResponse }
  | { type: "bootstrap-error"; message: string }
  | { type: "ws-ready"; snapshot: WireRoomSnapshot }
  | { type: "ws-interaction-updated"; snapshot: WireRoomSnapshot }
  | { type: "ws-chat-message"; message: WireChatMessage }
  | { type: "ws-ai-debug-turn-updated"; update: WireAIDebugTurnUpdate }
  | { type: "ws-typing"; participantId: string; active: boolean }
  | { type: "ws-connection"; state: HUDConnectionState }
  | { type: "mutation-snapshot"; snapshot: WireRoomSnapshot }
  | { type: "mutation-error"; message: string }
  | { type: "dismiss-mutation-error" }
  | { type: "set-tab"; tab: HUDNavbarTab }
  | { type: "set-viewer-typing"; active: boolean }
  | { type: "set-on-stage-draft"; value: string }
  | { type: "set-backstage-draft"; value: string }
  | { type: "set-side-chat-draft"; value: string }
  | { type: "set-sidebar-open"; open: boolean }
  | { type: "ai-debug-loading" }
  | { type: "ai-debug-loaded"; page: WireAIDebugTurnsPage; append: boolean }
  | { type: "ai-debug-error"; message: string }
  | { type: "ai-debug-toggle-turn"; turnId: string }
  | { type: "ai-debug-turn-loading"; turnId: string }
  | { type: "ai-debug-turn-loaded"; turn: WireAIDebugTurn };

type ComposerTypingSource = "on-stage" | "backstage" | "side-chat";

type ScenePlayerPostRequest = {
  scene_id: string;
  character_ids: string[];
  summary_text: string;
};

type SceneScopedRequest = {
  scene_id: string;
};

function initialState(): RuntimeState {
  return {
    phase: "loading",
    bootstrap: null,
    snapshot: null,
    chatMessages: [],
    remoteTypingParticipantIDs: [],
    viewerTyping: false,
    connectionState: "disconnected",
    activeTab: "on-stage",
    onStageDraft: "",
    backstageDraft: "",
    sideChatDraft: "",
    isSidebarOpen: false,
    aiDebug: { phase: "idle", turns: [], detailsByTurnId: {} },
  };
}

function snapshotFromBootstrap(bootstrap: BootstrapResponse): WireRoomSnapshot {
  return {
    interaction_state: bootstrap.interaction_state,
    participants: bootstrap.participants ?? [],
    character_inspection_catalog: bootstrap.character_inspection_catalog ?? {},
    chat: bootstrap.chat,
    latest_game_sequence: 0,
  };
}

function errorStatus(err: unknown): number | null {
  if (!err || typeof err !== "object") {
    return null;
  }
  const maybeStatus = (err as { status?: unknown }).status;
  return typeof maybeStatus === "number" ? maybeStatus : null;
}

function addTypingParticipant(ids: string[], participantId: string): string[] {
  return ids.includes(participantId) ? ids : [...ids, participantId];
}

function removeTypingParticipant(ids: string[], participantId: string): string[] {
  return ids.filter((id) => id !== participantId);
}

function upsertAIDebugTurnSummary(turns: WireAIDebugTurnSummary[], next: WireAIDebugTurnSummary): WireAIDebugTurnSummary[] {
  const index = turns.findIndex((turn) => turn.id === next.id);
  if (index === -1) {
    return [next, ...turns];
  }
  return turns.map((turn, turnIndex) => (turnIndex === index ? { ...turn, ...next } : turn));
}

function mergeAIDebugEntries(existing: WireAIDebugTurn["entries"], appended: WireAIDebugTurnUpdate["appended_entries"]): WireAIDebugTurn["entries"] {
  if (appended.length === 0) {
    return existing;
  }
  const bySequence = new Map(existing.map((entry) => [entry.sequence, entry]));
  for (const entry of appended) {
    bySequence.set(entry.sequence, entry);
  }
  return [...bySequence.values()].sort((left, right) => left.sequence - right.sequence);
}

function viewerParticipantID(
  bootstrap: BootstrapResponse | null,
  snapshot: WireRoomSnapshot | null,
): string {
  return snapshot?.interaction_state?.viewer?.participant_id
    ?? bootstrap?.interaction_state?.viewer?.participant_id
    ?? bootstrap?.viewer?.participant_id
    ?? "";
}

function reducer(state: RuntimeState, action: RuntimeAction): RuntimeState {
  switch (action.type) {
    case "bootstrap-loaded":
      return {
        ...state,
        phase: "ready",
        bootstrap: action.bootstrap,
        chatMessages: action.bootstrap.chat.messages,
        remoteTypingParticipantIDs: [],
      };
    case "runtime-resynced":
      return {
        ...state,
        phase: "ready",
        bootstrap: action.bootstrap,
        snapshot: snapshotFromBootstrap(action.bootstrap),
        chatMessages: action.bootstrap.chat.messages,
        remoteTypingParticipantIDs: [],
        mutationError: undefined,
      };
    case "bootstrap-error":
      return { ...state, phase: "error", errorMessage: action.message };
    case "ws-ready":
      return { ...state, snapshot: action.snapshot, remoteTypingParticipantIDs: [] };
    case "ws-interaction-updated":
      return { ...state, snapshot: action.snapshot, mutationError: undefined };
    case "ws-chat-message":
      return { ...state, chatMessages: [...state.chatMessages, action.message] };
    case "ws-ai-debug-turn-updated": {
      const turn = action.update.turn;
      if (!turn?.id) {
        return state;
      }
      const existingDetail = state.aiDebug.detailsByTurnId[turn.id];
      return {
        ...state,
        aiDebug: {
          ...state.aiDebug,
          turns: upsertAIDebugTurnSummary(state.aiDebug.turns, turn),
          detailsByTurnId: existingDetail
            ? {
              ...state.aiDebug.detailsByTurnId,
              [turn.id]: {
                ...existingDetail,
                ...turn,
                entries: mergeAIDebugEntries(existingDetail.entries, action.update.appended_entries ?? []),
              },
            }
            : state.aiDebug.detailsByTurnId,
        },
      };
    }
    case "ws-typing": {
      const participantId = action.participantId.trim();
      if (!participantId) {
        return state;
      }
      return {
        ...state,
        remoteTypingParticipantIDs: action.active
          ? addTypingParticipant(state.remoteTypingParticipantIDs, participantId)
          : removeTypingParticipant(state.remoteTypingParticipantIDs, participantId),
      };
    }
    case "ws-connection":
      return { ...state, connectionState: action.state };
    case "mutation-snapshot":
      return { ...state, snapshot: action.snapshot, mutationError: undefined };
    case "mutation-error":
      return { ...state, mutationError: action.message };
    case "dismiss-mutation-error":
      return { ...state, mutationError: undefined };
    case "set-tab":
      return { ...state, activeTab: action.tab };
    case "set-viewer-typing":
      return { ...state, viewerTyping: action.active };
    case "set-on-stage-draft":
      return { ...state, onStageDraft: action.value };
    case "set-backstage-draft":
      return { ...state, backstageDraft: action.value };
    case "set-side-chat-draft":
      return { ...state, sideChatDraft: action.value };
    case "set-sidebar-open":
      return { ...state, isSidebarOpen: action.open };
    case "ai-debug-loading":
      return {
        ...state,
        aiDebug: {
          ...state.aiDebug,
          phase: state.aiDebug.turns.length === 0 ? "loading" : state.aiDebug.phase,
          errorMessage: undefined,
        },
      };
    case "ai-debug-loaded":
      return {
        ...state,
        aiDebug: {
          ...state.aiDebug,
          phase: "ready",
          turns: action.append ? [...state.aiDebug.turns, ...action.page.turns] : action.page.turns,
          nextPageToken: action.page.next_page_token,
          errorMessage: undefined,
        },
      };
    case "ai-debug-error":
      return {
        ...state,
        aiDebug: {
          ...state.aiDebug,
          phase: "error",
          errorMessage: action.message,
        },
      };
    case "ai-debug-toggle-turn":
      return {
        ...state,
        aiDebug: {
          ...state.aiDebug,
          expandedTurnId: state.aiDebug.expandedTurnId === action.turnId ? undefined : action.turnId,
        },
      };
    case "ai-debug-turn-loading":
      return {
        ...state,
        aiDebug: {
          ...state.aiDebug,
          loadingTurnId: action.turnId,
          errorMessage: undefined,
        },
      };
    case "ai-debug-turn-loaded":
      return {
        ...state,
        aiDebug: {
          ...state.aiDebug,
          phase: "ready",
          loadingTurnId: state.aiDebug.loadingTurnId === action.turn.id ? undefined : state.aiDebug.loadingTurnId,
          detailsByTurnId: {
            ...state.aiDebug.detailsByTurnId,
            [action.turn.id]: action.turn,
          },
        },
      };
  }
}

function buildScenePlayerPostRequest(
  bootstrap: BootstrapResponse | null,
  snapshot: WireRoomSnapshot | null,
  draft: string,
): ScenePlayerPostRequest | null {
  const interactionState = snapshot?.interaction_state ?? bootstrap?.interaction_state;
  const viewerParticipantID = interactionState?.viewer?.participant_id ?? bootstrap?.viewer?.participant_id ?? "";
  const sceneID = interactionState?.active_scene?.scene_id?.trim() ?? "";
  const actingCharacterIDs = interactionState?.player_phase?.acting_character_ids ?? [];
  const sceneCharacters = interactionState?.active_scene?.characters ?? [];

  const viewerCharacterIDs = actingCharacterIDs.filter((characterID) =>
    sceneCharacters.some((character) =>
      character.character_id === characterID && character.owner_participant_id === viewerParticipantID,
    ),
  );

  if (!sceneID || viewerCharacterIDs.length === 0) {
    return null;
  }

  return {
    scene_id: sceneID,
    character_ids: viewerCharacterIDs,
    summary_text: draft,
  };
}

function buildSceneScopedRequest(
  bootstrap: BootstrapResponse | null,
  snapshot: WireRoomSnapshot | null,
): SceneScopedRequest | null {
  const sceneID = (snapshot?.interaction_state ?? bootstrap?.interaction_state)?.active_scene?.scene_id?.trim() ?? "";
  if (!sceneID) {
    return null;
  }
  return { scene_id: sceneID };
}

// --- Component ---

export function PlayRuntime({ shellConfig }: { shellConfig: PlayShellConfig }) {
  return (
    <TransitionPreferencesProvider>
      <PlayRuntimeInner shellConfig={shellConfig} />
    </TransitionPreferencesProvider>
  );
}

function PlayRuntimeInner({ shellConfig }: { shellConfig: PlayShellConfig }) {
  const [state, dispatch] = useReducer(reducer, undefined, initialState);
  const wsRef = useRef<WSConnection | null>(null);
  const typingSourceActiveRef = useRef<Record<ComposerTypingSource, boolean>>({
    "on-stage": false,
    backstage: false,
    "side-chat": false,
  });
  const typingSourceIdleTimersRef = useRef<Partial<Record<ComposerTypingSource, ReturnType<typeof setTimeout>>>>({});
  const typingHeartbeatRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const viewerTypingRef = useRef(false);
  const previousConnectionStateRef = useRef<HUDConnectionState>("disconnected");
  const {
    inspector,
    close: closeInspector,
    openForCharacter,
    openForParticipant,
    setActiveCharacter,
  } = usePlayerHUDCharacterInspector();
  const [settingsOpen, setSettingsOpen] = useState(false);
  const { preferences } = useTransitionPreferences();

  // Derive scene/interaction IDs for transition detection (before early returns so hooks are unconditional).
  const activeInteractionState = state.snapshot?.interaction_state ?? state.bootstrap?.interaction_state;
  const currentSceneId = activeInteractionState?.active_scene?.scene_id;
  const currentInteractionId = activeInteractionState?.active_scene?.current_interaction?.interaction_id;
  const { sceneTransitionKey, interactionTransitionActive, clearInteractionTransition } =
    useTransitionEffects(currentSceneId, currentInteractionId);

  const { playSceneSound, playInteractionSound } = useTransitionAudio({
    sceneUrl: state.bootstrap?.transition_sfx?.scene_change_url,
    interactionUrl: state.bootstrap?.transition_sfx?.interaction_change_url,
  });

  // Fire audio when transition keys change.
  const prevSceneKeyRef = useRef(0);
  useEffect(() => {
    if (sceneTransitionKey > 0 && sceneTransitionKey !== prevSceneKeyRef.current && preferences.sceneSound) {
      playSceneSound();
    }
    prevSceneKeyRef.current = sceneTransitionKey;
  }, [sceneTransitionKey, preferences.sceneSound, playSceneSound]);

  const prevInteractionActiveRef = useRef(false);
  useEffect(() => {
    if (interactionTransitionActive && !prevInteractionActiveRef.current && preferences.interactionSound) {
      playInteractionSound();
    }
    prevInteractionActiveRef.current = interactionTransitionActive;
  }, [interactionTransitionActive, preferences.interactionSound, playInteractionSound]);

  const typingTTLMs = Math.max(500, state.bootstrap?.realtime.typing_ttl_ms ?? 3_000);
  const aiDebugEnabled = state.bootstrap?.ai_debug_enabled ?? false;

  const clearTypingSourceIdleTimer = useCallback((source: ComposerTypingSource) => {
    const timer = typingSourceIdleTimersRef.current[source];
    if (timer) {
      clearTimeout(timer);
      delete typingSourceIdleTimersRef.current[source];
    }
  }, []);

  const stopTypingHeartbeat = useCallback(() => {
    if (typingHeartbeatRef.current) {
      clearInterval(typingHeartbeatRef.current);
      typingHeartbeatRef.current = null;
    }
  }, []);

  const sendTypingFrame = useCallback((active: boolean) => {
    wsRef.current?.send({
      type: FrameType.Typing,
      payload: { active },
    });
  }, []);

  const setViewerTyping = useCallback((active: boolean) => {
    if (viewerTypingRef.current === active) {
      return;
    }
    viewerTypingRef.current = active;
    dispatch({ type: "set-viewer-typing", active });
    if (active) {
      sendTypingFrame(true);
      if (!typingHeartbeatRef.current) {
        typingHeartbeatRef.current = setInterval(() => {
          if (viewerTypingRef.current) {
            sendTypingFrame(true);
          }
        }, Math.max(250, Math.floor(typingTTLMs / 2)));
      }
      return;
    }

    stopTypingHeartbeat();
    sendTypingFrame(false);
  }, [sendTypingFrame, stopTypingHeartbeat, typingTTLMs]);

  const syncViewerTypingFromSources = useCallback(() => {
    const active = Object.values(typingSourceActiveRef.current).some(Boolean);
    setViewerTyping(active);
  }, [setViewerTyping]);

  const setTypingSource = useCallback((source: ComposerTypingSource, active: boolean) => {
    clearTypingSourceIdleTimer(source);
    typingSourceActiveRef.current[source] = active;
    if (active) {
      typingSourceIdleTimersRef.current[source] = setTimeout(() => {
        typingSourceActiveRef.current[source] = false;
        delete typingSourceIdleTimersRef.current[source];
        syncViewerTypingFromSources();
      }, typingTTLMs);
    }
    syncViewerTypingFromSources();
  }, [clearTypingSourceIdleTimer, syncViewerTypingFromSources, typingTTLMs]);

  const clearAllTypingSources = useCallback(() => {
    (Object.keys(typingSourceActiveRef.current) as ComposerTypingSource[]).forEach((source) => {
      clearTypingSourceIdleTimer(source);
      typingSourceActiveRef.current[source] = false;
    });
    syncViewerTypingFromSources();
  }, [clearTypingSourceIdleTimer, syncViewerTypingFromSources]);

  const refreshRuntimeState = useCallback(async (): Promise<boolean> => {
    try {
      const bootstrap = await fetchBootstrap(shellConfig.bootstrapPath);
      dispatch({ type: "runtime-resynced", bootstrap });
      return true;
    } catch (err) {
      console.warn("[play runtime] failed to resync after mutation error", err);
      return false;
    }
  }, [shellConfig.bootstrapPath]);

  const loadAIDebugTurns = useCallback(async (pageToken?: string) => {
    dispatch({ type: "ai-debug-loading" });
    try {
      const page = await fetchAIDebugTurns(shellConfig.campaignId, pageToken);
      dispatch({ type: "ai-debug-loaded", page, append: Boolean(pageToken) });
    } catch (err) {
      dispatch({ type: "ai-debug-error", message: err instanceof Error ? err.message : "Failed to load AI debug turns" });
    }
  }, [shellConfig.campaignId]);

  const loadAIDebugTurn = useCallback(async (turnId: string) => {
    dispatch({ type: "ai-debug-turn-loading", turnId });
    try {
      const turn = await fetchAIDebugTurn(shellConfig.campaignId, turnId);
      dispatch({ type: "ai-debug-turn-loaded", turn });
    } catch (err) {
      dispatch({ type: "ai-debug-error", message: err instanceof Error ? err.message : "Failed to load AI debug turn" });
    }
  }, [shellConfig.campaignId]);

  // Bootstrap
  useEffect(() => {
    let cancelled = false;
    fetchBootstrap(shellConfig.bootstrapPath)
      .then((data) => {
        if (!cancelled) dispatch({ type: "bootstrap-loaded", bootstrap: data });
      })
      .catch((err) => {
        if (!cancelled) dispatch({ type: "bootstrap-error", message: String(err) });
      });
    return () => {
      cancelled = true;
    };
  }, [shellConfig.bootstrapPath]);

  // WebSocket — connect after bootstrap
  useEffect(() => {
    if (!state.bootstrap) return;
    const bootstrap = state.bootstrap;

    const conn = connectWebSocket({
      campaignId: bootstrap.campaign_id,
      lastGameSeq: 0,
      lastChatSeq: bootstrap.chat.latest_sequence_id,
      realtimeURL: shellConfig.realtimePath || "/realtime",
      onEvent: (event: WSEvent) => {
        switch (event.type) {
          case "ready":
            dispatch({ type: "ws-ready", snapshot: event.snapshot });
            break;
          case "interaction.updated":
            dispatch({ type: "ws-interaction-updated", snapshot: event.snapshot });
            break;
          case "chat.message":
            dispatch({ type: "ws-chat-message", message: event.message });
            break;
          case "ai-debug.turn.updated":
            dispatch({ type: "ws-ai-debug-turn-updated", update: event.update });
            break;
          case "typing":
            dispatch({ type: "ws-typing", participantId: event.participantId, active: event.active });
            break;
          case "connection":
            dispatch({ type: "ws-connection", state: event.state });
            break;
          case "resync":
            void refreshRuntimeState();
            break;
        }
      },
    });
    wsRef.current = conn;
    return () => {
      conn.close();
      wsRef.current = null;
    };
  }, [refreshRuntimeState, state.bootstrap, shellConfig.realtimePath]);

  useEffect(() => () => {
    clearAllTypingSources();
    stopTypingHeartbeat();
  }, [clearAllTypingSources, stopTypingHeartbeat]);

  useEffect(() => {
    if (!aiDebugEnabled || state.activeTab !== "ai-debug") {
      return;
    }
    if (state.aiDebug.phase === "idle") {
      void loadAIDebugTurns();
    }
  }, [aiDebugEnabled, loadAIDebugTurns, state.activeTab, state.aiDebug.phase]);

  useEffect(() => {
    if (!aiDebugEnabled || state.activeTab !== "ai-debug") {
      return;
    }
    const turnId = state.aiDebug.expandedTurnId;
    if (!turnId || state.aiDebug.detailsByTurnId[turnId]) {
      return;
    }
    void loadAIDebugTurn(turnId);
  }, [aiDebugEnabled, loadAIDebugTurn, state.activeTab, state.aiDebug.detailsByTurnId, state.aiDebug.expandedTurnId]);

  useEffect(() => {
    const previous = previousConnectionStateRef.current;
    previousConnectionStateRef.current = state.connectionState;
    if (!aiDebugEnabled || state.aiDebug.phase === "idle") {
      return;
    }
    if (state.connectionState !== "connected" || previous === "connected") {
      return;
    }
    void loadAIDebugTurns();
    if (state.aiDebug.expandedTurnId) {
      void loadAIDebugTurn(state.aiDebug.expandedTurnId);
    }
  }, [
    aiDebugEnabled,
    loadAIDebugTurn,
    loadAIDebugTurns,
    state.aiDebug.expandedTurnId,
    state.aiDebug.phase,
    state.connectionState,
  ]);

  // --- Handlers ---

  const campaignId = shellConfig.campaignId;

  const handleMutationSnapshot = useCallback((snapshot: WireRoomSnapshot) => {
    dispatch({ type: "mutation-snapshot", snapshot });
  }, []);

  const handleMutationError = useCallback((err: unknown) => {
    dispatch({ type: "mutation-error", message: err instanceof Error ? err.message : "Something went wrong" });
  }, []);

  const handleMutationFailure = useCallback((err: unknown) => {
    if (errorStatus(err) !== 409) {
      handleMutationError(err);
      return;
    }
    void refreshRuntimeState().then((refreshed) => {
      handleMutationError(new Error(
        refreshed
          ? "Scene state changed. The play view was refreshed."
          : err instanceof Error
            ? err.message
            : "Something went wrong",
      ));
    });
  }, [handleMutationError, refreshRuntimeState]);

  const handleOnStageSubmit = useCallback(() => {
    const draft = state.onStageDraft.trim();
    if (!draft) return;
    const request = buildScenePlayerPostRequest(state.bootstrap, state.snapshot, draft);
    if (!request) {
      handleMutationError(new Error("Scene context is missing for this action."));
      return;
    }
    dispatch({ type: "set-on-stage-draft", value: "" });
    setTypingSource("on-stage", false);
    mutations
      .submitScenePlayerPost(campaignId, request)
      .then(handleMutationSnapshot)
      .catch((err) => {
        dispatch({ type: "set-on-stage-draft", value: draft });
        setTypingSource("on-stage", draft.length > 0);
        handleMutationFailure(err);
      });
  }, [
    campaignId,
    handleMutationError,
    handleMutationFailure,
    handleMutationSnapshot,
    setTypingSource,
    state.bootstrap,
    state.onStageDraft,
    state.snapshot,
  ]);

  const handleOnStageSubmitAndYield = useCallback(() => {
    const draft = state.onStageDraft.trim();
    if (!draft) return;
    const request = buildScenePlayerPostRequest(state.bootstrap, state.snapshot, draft);
    if (!request) {
      handleMutationError(new Error("Scene context is missing for this action."));
      return;
    }
    dispatch({ type: "set-on-stage-draft", value: "" });
    setTypingSource("on-stage", false);
    mutations
      .submitScenePlayerPost(campaignId, request)
      .then(() => {
        const yieldRequest = buildSceneScopedRequest(state.bootstrap, state.snapshot);
        if (!yieldRequest) {
          throw new Error("Scene context is missing for this action.");
        }
        return mutations.yieldScenePlayerPhase(campaignId, yieldRequest);
      })
      .then(handleMutationSnapshot)
      .catch((err) => {
        dispatch({ type: "set-on-stage-draft", value: draft });
        setTypingSource("on-stage", draft.length > 0);
        handleMutationFailure(err);
      });
  }, [
    campaignId,
    handleMutationError,
    handleMutationFailure,
    handleMutationSnapshot,
    setTypingSource,
    state.bootstrap,
    state.onStageDraft,
    state.snapshot,
  ]);

  const handleOnStageYield = useCallback(() => {
    const request = buildSceneScopedRequest(state.bootstrap, state.snapshot);
    if (!request) {
      handleMutationError(new Error("Scene context is missing for this action."));
      return;
    }
    mutations.yieldScenePlayerPhase(campaignId, request).then(handleMutationSnapshot).catch(handleMutationFailure);
  }, [campaignId, handleMutationError, handleMutationFailure, handleMutationSnapshot, state.bootstrap, state.snapshot]);

  const handleOnStageUnyield = useCallback(() => {
    const request = buildSceneScopedRequest(state.bootstrap, state.snapshot);
    if (!request) {
      handleMutationError(new Error("Scene context is missing for this action."));
      return;
    }
    mutations.unyieldScenePlayerPhase(campaignId, request).then(handleMutationSnapshot).catch(handleMutationFailure);
  }, [campaignId, handleMutationError, handleMutationFailure, handleMutationSnapshot, state.bootstrap, state.snapshot]);

  const handleBackstageSend = useCallback(() => {
    const draft = state.backstageDraft.trim();
    if (!draft) return;
    dispatch({ type: "set-backstage-draft", value: "" });
    setTypingSource("backstage", false);
    mutations
      .postSessionOOC(campaignId, { body: draft })
      .then(handleMutationSnapshot)
      .catch((err) => {
        dispatch({ type: "set-backstage-draft", value: draft });
        setTypingSource("backstage", draft.length > 0);
        handleMutationFailure(err);
      });
  }, [campaignId, handleMutationFailure, handleMutationSnapshot, setTypingSource, state.backstageDraft]);

  const handleBackstageReadyToggle = useCallback(() => {
    if (!state.bootstrap) return;
    const fn = isViewerReadyToResume(state.bootstrap, state.snapshot)
      ? mutations.clearOOCReadyToResume
      : mutations.markOOCReadyToResume;
    fn(campaignId).then(handleMutationSnapshot).catch(handleMutationFailure);
  }, [campaignId, state.bootstrap, state.snapshot, handleMutationFailure, handleMutationSnapshot]);

  const handleSideChatSend = useCallback(() => {
    const draft = state.sideChatDraft.trim();
    if (!draft || !wsRef.current) return;
    wsRef.current.send({
      type: FrameType.ChatSend,
      payload: {
        client_message_id: crypto.randomUUID(),
        body: draft,
      },
    });
    dispatch({ type: "set-side-chat-draft", value: "" });
    setTypingSource("side-chat", false);
  }, [setTypingSource, state.sideChatDraft]);

  const handleAIDebugLoadMore = useCallback(() => {
    if (!state.aiDebug.nextPageToken) {
      return;
    }
    void loadAIDebugTurns(state.aiDebug.nextPageToken);
  }, [loadAIDebugTurns, state.aiDebug.nextPageToken]);

  const handleAIDebugToggleTurn = useCallback((turnId: string) => {
    dispatch({ type: "ai-debug-toggle-turn", turnId });
  }, []);

  // --- Render ---

  if (state.phase === "loading") {
    return (
      <main className="preview-shell">
        <div className="flex h-screen items-center justify-center">
          <span className="loading loading-spinner loading-lg" />
        </div>
      </main>
    );
  }

  if (state.phase === "error" || !state.bootstrap) {
    return (
      <main className="preview-shell">
        <div className="flex h-screen flex-col items-center justify-center gap-4">
          <h1 className="text-xl font-bold">Failed to load play session</h1>
          <p className="text-sm text-base-content/70">{state.errorMessage}</p>
          <button className="btn btn-primary" onClick={() => window.location.reload()}>
            Retry
          </button>
        </div>
      </main>
    );
  }

  const currentViewerParticipantID = viewerParticipantID(state.bootstrap, state.snapshot);
  const typingParticipantIDs = state.viewerTyping && currentViewerParticipantID
    ? addTypingParticipant(
        removeTypingParticipant(state.remoteTypingParticipantIDs, currentViewerParticipantID),
        currentViewerParticipantID,
      )
    : state.remoteTypingParticipantIDs;

  const hudState = mapToPlayerHUDState(
    state.bootstrap,
    state.snapshot,
    state.connectionState,
    state.activeTab,
    state.chatMessages,
    typingParticipantIDs,
    shellConfig.backURL,
  );

  function handleCharacterInspect(participantId: string, characterId: string) {
    const controller =
      hudState.campaignNavigation.characterControllers.find((entry) => entry.participantId === participantId)
      ?? hudState.campaignNavigation.characterControllers.find((entry) =>
        entry.characters.some((character) => character.id === characterId),
      );
    if (!controller) {
      console.warn("[play runtime inspect] missing character controller", { participantId, characterId });
      return;
    }
    console.info("[play runtime inspect] open character", {
      participantId,
      characterId,
      participantName: controller.participantName,
      characterCount: controller.characters.length,
    });
    openForCharacter(
      {
        name: controller.participantName,
        characters: controller.characters,
        isViewer: controller.isViewer,
      },
      characterId,
    );
  }

  function handleParticipantInspect(participantId: string) {
    const participant = state.activeTab === "on-stage"
      ? hudState.onStage.participants.find((entry) => entry.id === participantId)
      : state.activeTab === "backstage"
        ? hudState.backstage.participants.find((entry) => entry.id === participantId)
        : hudState.sideChat.participants.find((entry) => entry.id === participantId);
    const controller = hudState.campaignNavigation.characterControllers.find(
      (entry) => entry.participantId === participantId,
    );
    if (!participant) {
      console.warn("[play runtime inspect] missing participant", { participantId, activeTab: state.activeTab });
      return;
    }
    const viewerParticipantID = state.activeTab === "on-stage"
      ? hudState.onStage.viewerParticipantId
      : state.activeTab === "backstage"
        ? hudState.backstage.viewerParticipantId
        : hudState.sideChat.viewerParticipantId;
    console.info("[play runtime inspect] open participant", {
      participantId,
      participantName: participant.name,
      activeTab: state.activeTab,
      characterCount: controller?.characters.length || participant.characters.length,
    });
    openForParticipant({
      name: controller?.participantName || participant.name,
      characters: controller?.characters ?? participant.characters,
      isViewer: controller?.isViewer ?? (participant.id === viewerParticipantID),
    });
  }

  return (
    <>
        <PlayerHUDShell
          activeTab={hudState.activeTab}
          aiDebugEnabled={aiDebugEnabled}
          connectionState={hudState.connectionState}
        campaignNavigation={hudState.campaignNavigation}
        isSidebarOpen={state.isSidebarOpen}
        onSidebarOpenChange={(open) => dispatch({ type: "set-sidebar-open", open })}
        onTabChange={(tab) => dispatch({ type: "set-tab", tab })}
        onSettingsOpen={() => setSettingsOpen(true)}
        interactionTransitionActive={interactionTransitionActive}
        onInteractionTransitionEnd={clearInteractionTransition}
        onStage={hudState.onStage}
        onStageDraft={state.onStageDraft}
        onOnStageDraftChange={(value) => {
          dispatch({ type: "set-on-stage-draft", value });
          setTypingSource("on-stage", value.trim().length > 0);
        }}
        onOnStageSubmit={handleOnStageSubmit}
        onOnStageSubmitAndYield={handleOnStageSubmitAndYield}
        onOnStageYield={handleOnStageYield}
        onOnStageUnyield={handleOnStageUnyield}
        onCharacterInspect={handleCharacterInspect}
        onParticipantInspect={handleParticipantInspect}
        backstage={hudState.backstage}
        backstageDraft={state.backstageDraft}
        onBackstageDraftChange={(value) => {
          dispatch({ type: "set-backstage-draft", value });
          setTypingSource("backstage", value.trim().length > 0);
        }}
        onBackstageSend={handleBackstageSend}
        onBackstageReadyToggle={handleBackstageReadyToggle}
        sideChat={hudState.sideChat}
        sideChatDraft={state.sideChatDraft}
        onSideChatDraftChange={(value) => {
          dispatch({ type: "set-side-chat-draft", value });
          setTypingSource("side-chat", value.trim().length > 0);
        }}
        onSideChatSend={handleSideChatSend}
        aiDebug={state.aiDebug}
        onAIDebugLoadMore={handleAIDebugLoadMore}
        onAIDebugToggleTurn={handleAIDebugToggleTurn}
      />
      <PlayerHUDCharacterInspectorDialog
        isOpen={Boolean(inspector)}
        participantName={inspector?.participantName ?? ""}
        characters={inspector?.characters ?? []}
        activeCharacterId={inspector?.activeCharacterId}
        isViewer={inspector?.isViewer ?? false}
        characterInspectionCatalog={hudState.campaignNavigation.characterInspectionCatalog}
        onCharacterChange={setActiveCharacter}
        onClose={closeInspector}
      />
      <SceneTransitionOverlay transitionKey={sceneTransitionKey} />
      <TransitionSettingsModal isOpen={settingsOpen} onClose={() => setSettingsOpen(false)} />
      {state.mutationError && (
        <div className="toast toast-end toast-bottom z-50">
          <div role="alert" className="alert alert-error shadow-lg">
            <span className="text-sm">{state.mutationError}</span>
            <button
              className="btn btn-ghost btn-xs"
              onClick={() => dispatch({ type: "dismiss-mutation-error" })}
              aria-label="Dismiss error"
            >
              ✕
            </button>
          </div>
        </div>
      )}
    </>
  );
}
