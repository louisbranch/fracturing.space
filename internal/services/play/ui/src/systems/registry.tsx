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

export function resolveSystemRenderer(systemID: string, systemVersion: string): SystemRenderer {
  const key = `${systemID.trim().toLowerCase()}@${systemVersion.trim().toLowerCase()}`;
  return renderers.get(key) ?? baseRenderer;
}
