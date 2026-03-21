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
  return resp.json() as Promise<BootstrapResponse>;
}
