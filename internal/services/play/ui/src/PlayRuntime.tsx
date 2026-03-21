import { useCallback, useEffect, useReducer, useRef } from "react";
import { fetchBootstrap } from "./api/bootstrap";
import * as mutations from "./api/mutations";
import type { WSConnection, WSEvent } from "./api/websocket";
import { FrameType, connectWebSocket } from "./api/websocket";
import type { BootstrapResponse, WireChatMessage, WireRoomSnapshot } from "./api/types";
import type { HUDConnectionState, HUDNavbarTab } from "./interaction/player-hud/shared/contract";
import { PlayerHUDShell } from "./interaction/player-hud/player-hud-shell/PlayerHUDShell";
import {
  PlayerHUDCharacterInspectorDialog,
  usePlayerHUDCharacterInspector,
} from "./interaction/player-hud/shared/PlayerHUDCharacterInspector";
import { isViewerReadyToResume, mapToPlayerHUDState } from "./state/mapper";
import type { PlayShellConfig } from "./shell_config";

// --- State ---

type RuntimeState = {
  phase: "loading" | "ready" | "error";
  errorMessage?: string;
  mutationError?: string;
  bootstrap: BootstrapResponse | null;
  snapshot: WireRoomSnapshot | null;
  chatMessages: WireChatMessage[];
  connectionState: HUDConnectionState;
  activeTab: HUDNavbarTab;
  onStageDraft: string;
  backstageDraft: string;
  sideChatDraft: string;
  isSidebarOpen: boolean;
};

type RuntimeAction =
  | { type: "bootstrap-loaded"; bootstrap: BootstrapResponse }
  | { type: "runtime-resynced"; bootstrap: BootstrapResponse }
  | { type: "bootstrap-error"; message: string }
  | { type: "ws-ready"; snapshot: WireRoomSnapshot }
  | { type: "ws-interaction-updated"; snapshot: WireRoomSnapshot }
  | { type: "ws-chat-message"; message: WireChatMessage }
  | { type: "ws-connection"; state: HUDConnectionState }
  | { type: "mutation-snapshot"; snapshot: WireRoomSnapshot }
  | { type: "mutation-error"; message: string }
  | { type: "dismiss-mutation-error" }
  | { type: "set-tab"; tab: HUDNavbarTab }
  | { type: "set-on-stage-draft"; value: string }
  | { type: "set-backstage-draft"; value: string }
  | { type: "set-side-chat-draft"; value: string }
  | { type: "set-sidebar-open"; open: boolean };

type ScenePlayerPostRequest = {
  scene_id: string;
  character_ids: string[];
  summary_text: string;
  yield_after_post?: boolean;
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
    connectionState: "disconnected",
    activeTab: "on-stage",
    onStageDraft: "",
    backstageDraft: "",
    sideChatDraft: "",
    isSidebarOpen: false,
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

function reducer(state: RuntimeState, action: RuntimeAction): RuntimeState {
  switch (action.type) {
    case "bootstrap-loaded":
      return {
        ...state,
        phase: "ready",
        bootstrap: action.bootstrap,
        chatMessages: action.bootstrap.chat.messages,
      };
    case "runtime-resynced":
      return {
        ...state,
        phase: "ready",
        bootstrap: action.bootstrap,
        snapshot: snapshotFromBootstrap(action.bootstrap),
        chatMessages: action.bootstrap.chat.messages,
        mutationError: undefined,
      };
    case "bootstrap-error":
      return { ...state, phase: "error", errorMessage: action.message };
    case "ws-ready":
      return { ...state, snapshot: action.snapshot };
    case "ws-interaction-updated":
      return { ...state, snapshot: action.snapshot, mutationError: undefined };
    case "ws-chat-message":
      return { ...state, chatMessages: [...state.chatMessages, action.message] };
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
    case "set-on-stage-draft":
      return { ...state, onStageDraft: action.value };
    case "set-backstage-draft":
      return { ...state, backstageDraft: action.value };
    case "set-side-chat-draft":
      return { ...state, sideChatDraft: action.value };
    case "set-sidebar-open":
      return { ...state, isSidebarOpen: action.open };
  }
}

function buildScenePlayerPostRequest(
  bootstrap: BootstrapResponse | null,
  snapshot: WireRoomSnapshot | null,
  draft: string,
  yieldAfterPost = false,
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
    yield_after_post: yieldAfterPost ? true : undefined,
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
  const [state, dispatch] = useReducer(reducer, undefined, initialState);
  const wsRef = useRef<WSConnection | null>(null);
  const {
    inspector,
    close: closeInspector,
    openForCharacter,
    openForParticipant,
    setActiveCharacter,
  } = usePlayerHUDCharacterInspector();

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
    mutations
      .submitScenePlayerPost(campaignId, request)
      .then(handleMutationSnapshot)
      .catch((err) => {
        dispatch({ type: "set-on-stage-draft", value: draft });
        handleMutationFailure(err);
      });
  }, [campaignId, handleMutationError, handleMutationFailure, handleMutationSnapshot, state.bootstrap, state.onStageDraft, state.snapshot]);

  const handleOnStageSubmitAndYield = useCallback(() => {
    const draft = state.onStageDraft.trim();
    if (!draft) return;
    const request = buildScenePlayerPostRequest(state.bootstrap, state.snapshot, draft, true);
    if (!request) {
      handleMutationError(new Error("Scene context is missing for this action."));
      return;
    }
    dispatch({ type: "set-on-stage-draft", value: "" });
    mutations
      .submitScenePlayerPost(campaignId, request)
      .then(handleMutationSnapshot)
      .catch((err) => {
        dispatch({ type: "set-on-stage-draft", value: draft });
        handleMutationFailure(err);
      });
  }, [campaignId, handleMutationError, handleMutationFailure, handleMutationSnapshot, state.bootstrap, state.onStageDraft, state.snapshot]);

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
    mutations
      .postSessionOOC(campaignId, { body: draft })
      .then(handleMutationSnapshot)
      .catch((err) => {
        dispatch({ type: "set-backstage-draft", value: draft });
        handleMutationFailure(err);
      });
  }, [campaignId, state.backstageDraft, handleMutationFailure, handleMutationSnapshot]);

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
  }, [state.sideChatDraft]);

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

  const hudState = mapToPlayerHUDState(
    state.bootstrap,
    state.snapshot,
    state.connectionState,
    state.activeTab,
    state.chatMessages,
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
        connectionState={hudState.connectionState}
        campaignNavigation={hudState.campaignNavigation}
        isSidebarOpen={state.isSidebarOpen}
        onSidebarOpenChange={(open) => dispatch({ type: "set-sidebar-open", open })}
        onTabChange={(tab) => dispatch({ type: "set-tab", tab })}
        onStage={hudState.onStage}
        onStageDraft={state.onStageDraft}
        onOnStageDraftChange={(value) => dispatch({ type: "set-on-stage-draft", value })}
        onOnStageSubmit={handleOnStageSubmit}
        onOnStageSubmitAndYield={handleOnStageSubmitAndYield}
        onOnStageYield={handleOnStageYield}
        onOnStageUnyield={handleOnStageUnyield}
        onCharacterInspect={handleCharacterInspect}
        onParticipantInspect={handleParticipantInspect}
        backstage={hudState.backstage}
        backstageDraft={state.backstageDraft}
        onBackstageDraftChange={(value) => dispatch({ type: "set-backstage-draft", value })}
        onBackstageSend={handleBackstageSend}
        onBackstageReadyToggle={handleBackstageReadyToggle}
        sideChat={hudState.sideChat}
        sideChatDraft={state.sideChatDraft}
        onSideChatDraftChange={(value) => dispatch({ type: "set-side-chat-draft", value })}
        onSideChatSend={handleSideChatSend}
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
