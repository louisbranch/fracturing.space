import { useCallback, useEffect, useReducer, useRef } from "react";
import { fetchBootstrap } from "./api/bootstrap";
import * as mutations from "./api/mutations";
import type { WSConnection, WSEvent } from "./api/websocket";
import { FrameType, connectWebSocket } from "./api/websocket";
import type { BootstrapResponse, WireChatMessage, WireRoomSnapshot } from "./api/types";
import type { HUDConnectionState, HUDNavbarTab } from "./interaction/player-hud/shared/contract";
import { PlayerHUDShell } from "./interaction/player-hud/player-hud-shell/PlayerHUDShell";
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
  | { type: "bootstrap-error"; message: string }
  | { type: "ws-ready"; snapshot: WireRoomSnapshot }
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

function reducer(state: RuntimeState, action: RuntimeAction): RuntimeState {
  switch (action.type) {
    case "bootstrap-loaded":
      return {
        ...state,
        phase: "ready",
        bootstrap: action.bootstrap,
        chatMessages: action.bootstrap.chat.messages,
      };
    case "bootstrap-error":
      return { ...state, phase: "error", errorMessage: action.message };
    case "ws-ready":
      return { ...state, snapshot: action.snapshot };
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

// --- Component ---

export function PlayRuntime({ shellConfig }: { shellConfig: PlayShellConfig }) {
  const [state, dispatch] = useReducer(reducer, undefined, initialState);
  const wsRef = useRef<WSConnection | null>(null);

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
          case "chat.message":
            dispatch({ type: "ws-chat-message", message: event.message });
            break;
          case "connection":
            dispatch({ type: "ws-connection", state: event.state });
            break;
        }
      },
    });
    wsRef.current = conn;
    return () => {
      conn.close();
      wsRef.current = null;
    };
  }, [state.bootstrap, shellConfig.realtimePath]);

  // --- Handlers ---

  const campaignId = shellConfig.campaignId;

  const handleMutationSnapshot = useCallback((snapshot: WireRoomSnapshot) => {
    dispatch({ type: "mutation-snapshot", snapshot });
  }, []);

  const handleMutationError = useCallback((err: unknown) => {
    dispatch({ type: "mutation-error", message: err instanceof Error ? err.message : "Something went wrong" });
  }, []);

  const handleOnStageSubmit = useCallback(() => {
    const draft = state.onStageDraft.trim();
    if (!draft) return;
    dispatch({ type: "set-on-stage-draft", value: "" });
    mutations
      .submitScenePlayerPost(campaignId, { summary_text: draft })
      .then(handleMutationSnapshot)
      .catch((err) => {
        dispatch({ type: "set-on-stage-draft", value: draft });
        handleMutationError(err);
      });
  }, [campaignId, state.onStageDraft, handleMutationSnapshot, handleMutationError]);

  const handleOnStageSubmitAndYield = useCallback(() => {
    const draft = state.onStageDraft.trim();
    if (!draft) return;
    dispatch({ type: "set-on-stage-draft", value: "" });
    mutations
      .submitScenePlayerPost(campaignId, { summary_text: draft })
      .then((snap) => {
        handleMutationSnapshot(snap);
        return mutations.yieldScenePlayerPhase(campaignId);
      })
      .then(handleMutationSnapshot)
      .catch((err) => {
        dispatch({ type: "set-on-stage-draft", value: draft });
        handleMutationError(err);
      });
  }, [campaignId, state.onStageDraft, handleMutationSnapshot, handleMutationError]);

  const handleOnStageYield = useCallback(() => {
    mutations.yieldScenePlayerPhase(campaignId).then(handleMutationSnapshot).catch(handleMutationError);
  }, [campaignId, handleMutationSnapshot, handleMutationError]);

  const handleOnStageUnyield = useCallback(() => {
    mutations.unyieldScenePlayerPhase(campaignId).then(handleMutationSnapshot).catch(handleMutationError);
  }, [campaignId, handleMutationSnapshot, handleMutationError]);

  const handleBackstageSend = useCallback(() => {
    const draft = state.backstageDraft.trim();
    if (!draft) return;
    dispatch({ type: "set-backstage-draft", value: "" });
    mutations
      .postSessionOOC(campaignId, { body: draft })
      .then(handleMutationSnapshot)
      .catch((err) => {
        dispatch({ type: "set-backstage-draft", value: draft });
        handleMutationError(err);
      });
  }, [campaignId, state.backstageDraft, handleMutationSnapshot, handleMutationError]);

  const handleBackstageReadyToggle = useCallback(() => {
    if (!state.bootstrap) return;
    const fn = isViewerReadyToResume(state.bootstrap, state.snapshot)
      ? mutations.clearOOCReadyToResume
      : mutations.markOOCReadyToResume;
    fn(campaignId).then(handleMutationSnapshot).catch(handleMutationError);
  }, [campaignId, state.bootstrap, state.snapshot, handleMutationSnapshot, handleMutationError]);

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
  );

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
