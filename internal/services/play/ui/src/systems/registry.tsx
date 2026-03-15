import type { SystemSummary } from "../protocol";
import type { SystemRenderer, SystemRendererProps } from "../types";
import { BaseGameView } from "./base/BaseGameView";
import { DaggerheartGameView } from "./daggerheart/DaggerheartGameView";

const baseRenderer: SystemRenderer = {
  id: "base",
  render(props: SystemRendererProps) {
    return <BaseGameView {...props} />;
  },
};

const daggerheartRenderer: SystemRenderer = {
  id: "daggerheart@v1",
  render(props: SystemRendererProps) {
    return <DaggerheartGameView {...props} />;
  },
};

const renderers = new Map<string, SystemRenderer>([[daggerheartRenderer.id, daggerheartRenderer]]);

export function resolveSystemRenderer(system: Pick<SystemSummary, "id" | "version">): SystemRenderer {
  const key = rendererKey(system.id, system.version);
  return renderers.get(key) ?? baseRenderer;
}

function rendererKey(systemID: string, systemVersion: string): string {
  return `${systemID.trim().toLowerCase()}@${normalizeRendererVersion(systemVersion)}`;
}

function normalizeRendererVersion(version: string): string {
  const normalized = version.trim().toLowerCase();
  const majorVersion = normalized.match(/^v?(\d+)(?:\.\d+){0,2}(?:[-+].+)?$/);
  if (majorVersion?.[1]) {
    return `v${majorVersion[1]}`;
  }
  return normalized;
}
