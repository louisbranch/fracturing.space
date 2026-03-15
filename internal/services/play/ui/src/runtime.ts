import { useEffect, useEffectEvent, useRef, useState } from "react";
import {
  PlayRealtimeClient,
  type RealtimeStatus,
  type ServerFrame,
} from "./realtime";
import type { PlayBootstrap, PlayRoomSnapshot } from "./protocol";
import {
  applyHistoryFailure,
  applyHistoryLoaded,
  applyHistoryLoadStarted,
  applyRealtimeFrame,
  applyRealtimeStatus,
  createFailedRuntimeState,
  createLoadedRuntimeState,
  initialRuntimeState,
  type RuntimeState,
} from "./runtime_state";
import { errorMessage, fetchBootstrapData, fetchHistoryPage } from "./runtime_transport";
import { resolveCampaignId } from "./utils";

export function usePlayRuntime(): {
  state: RuntimeState;
  campaignId: string;
  loadOlderMessages: () => Promise<void>;
  sendChat: (body: string) => void;
  setChatTyping: (active: boolean) => void;
  setDraftTyping: (active: boolean) => void;
} {
  const campaignId = resolveCampaignId(window.location.pathname);
  const [state, setState] = useState<RuntimeState>(initialRuntimeState);
  const realtimeRef = useRef<PlayRealtimeClient | null>(null);

  const onFrame = useEffectEvent((frame: ServerFrame) => {
    setState((current) => applyRealtimeFrame(current, frame));
  });

  const connectRealtime = useEffectEvent((bootstrap: PlayBootstrap, snapshot: PlayRoomSnapshot) => {
    const previous = realtimeRef.current;
    realtimeRef.current = null;
    previous?.close();

    let client: PlayRealtimeClient | null = null;
    const handleStatus = (status: RealtimeStatus) => {
      if (realtimeRef.current !== client) {
        return;
      }
      if (status.type !== "open") {
        realtimeRef.current = null;
      }
      setState((current) => applyRealtimeStatus(current, status));
    };

    client = new PlayRealtimeClient(
      bootstrap,
      onFrame,
      handleStatus,
      snapshot.latest_game_sequence,
      snapshot.chat?.latest_sequence_id ?? bootstrap.chat.latest_sequence_id,
    );
    realtimeRef.current = client;
  });

  const loadHistory = useEffectEvent(async (historyURL?: string, beforeSequence?: number) => {
    setState((current) => applyHistoryLoadStarted(current));
    try {
      const messages = await fetchHistoryPage(campaignId, historyURL, beforeSequence);
      setState((current) => applyHistoryLoaded(current, { messages }));
    } catch (error) {
      const message = errorMessage(error, "failed to load chat history");
      setState((current) => applyHistoryFailure(current, message));
    }
  });

  useEffect(() => {
    let cancelled = false;

    void (async () => {
      try {
        const runtimeData = await fetchBootstrapData(campaignId);
        if (cancelled) {
          return;
        }

        setState(createLoadedRuntimeState(runtimeData));

        if (cancelled) {
          return;
        }

        try {
          connectRealtime(runtimeData.bootstrap, runtimeData.snapshot);
        } catch (error) {
          if (cancelled) {
            return;
          }
          setState((current) =>
            applyRealtimeStatus(current, {
              type: "disconnected",
              message: errorMessage(error, "failed to connect realtime"),
            }),
          );
        }
      } catch (error) {
        if (cancelled) {
          return;
        }
        setState(createFailedRuntimeState(errorMessage(error, "failed to load active play")));
      }
    })();

    return () => {
      cancelled = true;
      const client = realtimeRef.current;
      realtimeRef.current = null;
      client?.close();
    };
  }, [campaignId]);

  return {
    state,
    campaignId,
    loadOlderMessages: async () => {
      if (!state.bootstrap || state.loadingHistory) {
        return;
      }
      const firstSequence = state.messages[0]?.sequence_id ?? state.snapshot?.chat?.latest_sequence_id ?? 0;
      if (firstSequence <= 1) {
        return;
      }
      await loadHistory(state.snapshot?.chat?.history_url, firstSequence);
    },
    sendChat: (body) => {
      realtimeRef.current?.sendChat(body);
    },
    setChatTyping: (active) => {
      realtimeRef.current?.sendChatTyping(active);
    },
    setDraftTyping: (active) => {
      realtimeRef.current?.sendDraftTyping(active);
    },
  };
}
