import { useEffect, useEffectEvent, useRef, useState } from "react";
import { PlayRealtimeClient, type ServerFrame } from "./realtime";
import type {
  PlayBootstrap,
  PlayChatMessage,
  PlayRoomSnapshot,
  TypingEvent,
} from "./types";
import {
  mergeMessages,
  normalizeBootstrap,
  normalizeHistory,
  normalizeSnapshot,
  resolveCampaignId,
} from "./utils";

type TypingState = Record<string, TypingEvent>;

type RuntimeState = {
  bootstrap?: PlayBootstrap;
  snapshot?: PlayRoomSnapshot;
  messages: PlayChatMessage[];
  chatTyping: TypingState;
  draftTyping: TypingState;
  error: string;
  loading: boolean;
  connected: boolean;
  loadingHistory: boolean;
};

const initialState: RuntimeState = {
  messages: [],
  chatTyping: {},
  draftTyping: {},
  error: "",
  loading: true,
  connected: false,
  loadingHistory: false,
};

export function usePlayRuntime(): {
  state: RuntimeState;
  campaignId: string;
  loadOlderMessages: () => Promise<void>;
  sendChat: (body: string) => void;
  setChatTyping: (active: boolean) => void;
  setDraftTyping: (active: boolean) => void;
} {
  const campaignId = resolveCampaignId(window.location.pathname);
  const [state, setState] = useState<RuntimeState>(initialState);
  const realtimeRef = useRef<PlayRealtimeClient | null>(null);

  const onFrame = useEffectEvent((frame: ServerFrame) => {
    switch (frame.type) {
      case "play.ready":
      case "play.interaction.updated":
        setState((current) => {
          if (!current.bootstrap) {
            return current;
          }
          const snapshot = normalizeSnapshot(current.bootstrap, frame.payload, current.snapshot?.chat);
          return {
            ...current,
            snapshot,
            connected: true,
          };
        });
        return;
      case "play.chat.message":
        setState((current) => ({
          ...current,
          messages: mergeMessages(current.messages, [frame.payload.message]),
        }));
        return;
      case "play.chat.typing":
        setState((current) => ({
          ...current,
          chatTyping: updateTypingState(current.chatTyping, frame.payload),
        }));
        return;
      case "play.draft.typing":
        setState((current) => ({
          ...current,
          draftTyping: updateTypingState(current.draftTyping, frame.payload),
        }));
        return;
      case "play.error":
        setState((current) => ({
          ...current,
          error: frame.payload.message,
        }));
        return;
      case "play.resync":
        setState((current) => ({
          ...current,
          error: frame.payload.reason,
        }));
        return;
    }
  });

  const fetchBootstrap = useEffectEvent(async () => {
    const response = await fetch(`/api/campaigns/${campaignId}/bootstrap`, {
      credentials: "same-origin",
      headers: {
        Accept: "application/json",
      },
    });

    if (!response.ok) {
      throw new Error(`bootstrap request failed: ${response.status}`);
    }

    const payload = normalizeBootstrap((await response.json()) as PlayBootstrap);
    setState((current) => ({
      ...current,
      bootstrap: payload,
      snapshot: normalizeSnapshot(payload),
      loading: false,
      error: "",
    }));
    await loadHistory(payload.chat.history_url, payload.chat.latest_sequence_id + 1);
    connectRealtime(payload);
  });

  const connectRealtime = useEffectEvent((bootstrap: PlayBootstrap) => {
    realtimeRef.current?.close();
    const client = new PlayRealtimeClient(
      bootstrap,
      onFrame,
      () => {
        setState((current) => ({ ...current, connected: true }));
      },
      (message) => {
        setState((current) => ({ ...current, connected: false, error: message }));
      },
      state.snapshot?.latest_game_sequence ?? 0,
      state.snapshot?.chat?.latest_sequence_id ?? bootstrap.chat.latest_sequence_id,
    );
    realtimeRef.current = client;
  });

  const loadHistory = useEffectEvent(async (historyURL?: string, beforeSequence?: number) => {
    const url = new URL(historyURL ?? `/api/campaigns/${campaignId}/chat/history`, window.location.origin);
    if (beforeSequence !== undefined) {
      url.searchParams.set("before_seq", String(beforeSequence));
    }
    url.searchParams.set("limit", "25");

    setState((current) => ({ ...current, loadingHistory: true }));
    const response = await fetch(url, {
      credentials: "same-origin",
      headers: {
        Accept: "application/json",
      },
    });
    if (!response.ok) {
      throw new Error(`history request failed: ${response.status}`);
    }
    const payload = await response.json();
    const messages = normalizeHistory(payload);
    setState((current) => ({
      ...current,
      loadingHistory: false,
      messages: mergeMessages(current.messages, messages),
    }));
  });

  useEffect(() => {
    let cancelled = false;

    fetchBootstrap().catch((error) => {
      if (cancelled) {
        return;
      }
      const message = error instanceof Error ? error.message : "failed to load active play";
      setState((current) => ({
        ...current,
        loading: false,
        error: message,
      }));
    });

    return () => {
      cancelled = true;
      realtimeRef.current?.close();
      realtimeRef.current = null;
    };
  }, [campaignId]);

  return {
    state,
    campaignId,
    loadOlderMessages: async () => {
      if (!state.bootstrap) {
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

function updateTypingState(current: TypingState, event: TypingEvent): TypingState {
  const next = { ...current };
  if (!event.active) {
    delete next[event.participant_id];
    return next;
  }
  next[event.participant_id] = event;
  return next;
}
