import type {
  PlayBootstrap,
  PlayChatMessage,
} from "./protocol";
import {
  normalizeBootstrap,
  normalizeHistory,
  normalizeSnapshot,
} from "./utils";
import type { RuntimeBootstrapData } from "./runtime_state";

export async function fetchBootstrapData(campaignId: string): Promise<RuntimeBootstrapData> {
  const response = await fetch(`/api/campaigns/${campaignId}/bootstrap`, {
    credentials: "same-origin",
    headers: {
      Accept: "application/json",
    },
  });

  if (!response.ok) {
    throw new Error(`bootstrap request failed: ${response.status}`);
  }

  const bootstrap = normalizeBootstrap((await response.json()) as PlayBootstrap);
  const snapshot = normalizeSnapshot(bootstrap);
  const messages = normalizeHistory({
    session_id: bootstrap.chat.session_id,
    messages: bootstrap.chat.messages,
  });

  return {
    bootstrap,
    snapshot,
    messages,
  };
}

export async function fetchHistoryPage(
  campaignId: string,
  historyURL?: string,
  beforeSequence?: number,
): Promise<PlayChatMessage[]> {
  const url = new URL(historyURL ?? `/api/campaigns/${campaignId}/chat/history`, window.location.origin);
  if (beforeSequence !== undefined) {
    url.searchParams.set("before_seq", String(beforeSequence));
  }
  url.searchParams.set("limit", "25");

  const response = await fetch(url, {
    credentials: "same-origin",
    headers: {
      Accept: "application/json",
    },
  });
  if (!response.ok) {
    throw new Error(`history request failed: ${response.status}`);
  }
  const payload = (await response.json()) as { session_id?: string; messages?: PlayChatMessage[] };
  return normalizeHistory({ session_id: payload.session_id ?? "", messages: payload.messages ?? [] });
}

export function errorMessage(error: unknown, fallback: string): string {
  return error instanceof Error ? error.message : fallback;
}
