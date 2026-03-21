import type { BootstrapResponse } from "./types";

export async function fetchBootstrap(path: string): Promise<BootstrapResponse> {
  const resp = await fetch(path, {
    credentials: "same-origin",
    headers: { Accept: "application/json" },
  });
  if (!resp.ok) {
    throw new Error(`bootstrap failed: ${resp.status}`);
  }
  return resp.json() as Promise<BootstrapResponse>;
}
