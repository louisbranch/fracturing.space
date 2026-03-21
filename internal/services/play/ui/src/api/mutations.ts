import type { WireRoomSnapshot } from "./types";

function interactionURL(campaignId: string, action: string): string {
  return `/api/campaigns/${encodeURIComponent(campaignId)}/interaction/${action}`;
}

async function postInteraction(
  campaignId: string,
  action: string,
  body?: Record<string, unknown>,
): Promise<WireRoomSnapshot> {
  const resp = await fetch(interactionURL(campaignId, action), {
    method: "POST",
    credentials: "same-origin",
    headers: { "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!resp.ok) {
    const text = await resp.text().catch(() => "");
    throw new Error(`mutation ${action} failed: ${resp.status} ${text}`);
  }
  return resp.json() as Promise<WireRoomSnapshot>;
}

export function submitScenePlayerPost(campaignId: string, body: { summary_text: string }): Promise<WireRoomSnapshot> {
  return postInteraction(campaignId, "submit-scene-player-post", body);
}

export function yieldScenePlayerPhase(campaignId: string): Promise<WireRoomSnapshot> {
  return postInteraction(campaignId, "yield-scene-player-phase");
}

export function unyieldScenePlayerPhase(campaignId: string): Promise<WireRoomSnapshot> {
  return postInteraction(campaignId, "unyield-scene-player-phase");
}

export function postSessionOOC(campaignId: string, body: { body: string }): Promise<WireRoomSnapshot> {
  return postInteraction(campaignId, "post-session-ooc", body);
}

export function markOOCReadyToResume(campaignId: string): Promise<WireRoomSnapshot> {
  return postInteraction(campaignId, "mark-ooc-ready-to-resume");
}

export function clearOOCReadyToResume(campaignId: string): Promise<WireRoomSnapshot> {
  return postInteraction(campaignId, "clear-ooc-ready-to-resume");
}
