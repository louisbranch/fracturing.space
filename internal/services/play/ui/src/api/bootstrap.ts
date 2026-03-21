import type { BootstrapResponse } from "./types";

const BOOTSTRAP_TIMEOUT_MS = 10_000;

export async function fetchBootstrap(path: string): Promise<BootstrapResponse> {
  const resp = await fetch(path, {
    credentials: "same-origin",
    headers: { Accept: "application/json" },
    signal: AbortSignal.timeout(BOOTSTRAP_TIMEOUT_MS),
  });
  if (!resp.ok) {
    throw new Error(`bootstrap failed: ${resp.status}`);
  }
  const payload = await resp.json() as BootstrapResponse;
  console.info("[play bootstrap]", {
    path,
    participants: payload.participants?.length ?? 0,
    characterCatalogEntries: Object.keys(payload.character_inspection_catalog ?? {}).length,
    activeSessionId: payload.interaction_state?.active_session?.session_id ?? "",
  });
  return payload;
}
