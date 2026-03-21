import type { WireRoomSnapshot } from "./types";

const MUTATION_TIMEOUT_MS = 15_000;

type InteractionErrorPayload = {
  error?: string;
};

export class InteractionRequestError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "InteractionRequestError";
    this.status = status;
  }
}

function interactionURL(campaignId: string, action: string): string {
  return `/api/campaigns/${encodeURIComponent(campaignId)}/interaction/${action}`;
}

async function readInteractionErrorMessage(resp: Response): Promise<string> {
  try {
    const payload = await resp.json() as InteractionErrorPayload;
    return payload.error?.trim() ?? "";
  } catch {
    return "";
  }
}

async function postInteraction(
  campaignId: string,
  action: string,
  body?: Record<string, unknown>,
): Promise<WireRoomSnapshot> {
  console.info("[play mutation request]", { campaignId, action, body });
  const resp = await fetch(interactionURL(campaignId, action), {
    method: "POST",
    credentials: "same-origin",
    headers: { "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
    signal: AbortSignal.timeout(MUTATION_TIMEOUT_MS),
  });
  if (!resp.ok) {
    const message = await readInteractionErrorMessage(resp);
    throw new InteractionRequestError(
      resp.status,
      message || `Action failed (${resp.status}). Please try again.`,
    );
  }
  return resp.json() as Promise<WireRoomSnapshot>;
}

export function submitScenePlayerPost(campaignId: string, body: {
  scene_id: string;
  character_ids: string[];
  summary_text: string;
}): Promise<WireRoomSnapshot> {
  return postInteraction(campaignId, "submit-scene-player-action", body);
}

export function yieldScenePlayerPhase(campaignId: string, body: { scene_id: string }): Promise<WireRoomSnapshot> {
  return postInteraction(campaignId, "yield-scene-player-phase", body);
}

export function unyieldScenePlayerPhase(campaignId: string, body: { scene_id: string }): Promise<WireRoomSnapshot> {
  return postInteraction(campaignId, "withdraw-scene-player-yield", body);
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
