import type { WireAIDebugTurn, WireAIDebugTurnsPage } from "./types";

const AI_DEBUG_TIMEOUT_MS = 10_000;

function aiDebugTurnsURL(campaignId: string, pageToken?: string): string {
  const url = new URL(`/api/campaigns/${encodeURIComponent(campaignId)}/ai-debug/turns`, window.location.origin);
  url.searchParams.set("page_size", "20");
  if (pageToken?.trim()) {
    url.searchParams.set("page_token", pageToken.trim());
  }
  return url.pathname + url.search;
}

function aiDebugTurnURL(campaignId: string, turnId: string): string {
  return `/api/campaigns/${encodeURIComponent(campaignId)}/ai-debug/turns/${encodeURIComponent(turnId)}`;
}

export async function fetchAIDebugTurns(campaignId: string, pageToken?: string): Promise<WireAIDebugTurnsPage> {
  const resp = await fetch(aiDebugTurnsURL(campaignId, pageToken), {
    credentials: "same-origin",
    headers: { Accept: "application/json" },
    signal: AbortSignal.timeout(AI_DEBUG_TIMEOUT_MS),
  });
  if (!resp.ok) {
    throw new Error(`ai debug turns failed: ${resp.status}`);
  }
  return await resp.json() as WireAIDebugTurnsPage;
}

export async function fetchAIDebugTurn(campaignId: string, turnId: string): Promise<WireAIDebugTurn> {
  const resp = await fetch(aiDebugTurnURL(campaignId, turnId), {
    credentials: "same-origin",
    headers: { Accept: "application/json" },
    signal: AbortSignal.timeout(AI_DEBUG_TIMEOUT_MS),
  });
  if (!resp.ok) {
    throw new Error(`ai debug turn failed: ${resp.status}`);
  }
  return await resp.json() as WireAIDebugTurn;
}
