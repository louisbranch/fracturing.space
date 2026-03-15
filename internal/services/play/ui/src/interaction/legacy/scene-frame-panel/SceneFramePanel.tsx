import { PhaseStatusBanner } from "../phase-status-banner/PhaseStatusBanner";
import type { SceneFramePanelProps } from "./contract";

// SceneFramePanel anchors the current beat around the active scene, prior GM
// output, and current player-facing frame text.
export function SceneFramePanel({ phase, scene }: SceneFramePanelProps) {
  return (
    <section className="preview-panel" aria-label="Active scene frame">
      <div className="preview-panel-body gap-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <span className="preview-kicker">Scene</span>
            <h2 className="font-display text-3xl text-base-content">
              {scene?.name?.trim() || "No active scene"}
            </h2>
            {scene?.description ? (
              <p className="mt-2 max-w-3xl text-sm leading-6 text-base-content/72">{scene.description}</p>
            ) : (
              <p className="mt-2 text-sm text-base-content/60">
                The session is active, but no authoritative scene has been selected yet.
              </p>
            )}
          </div>
        </div>

        <PhaseStatusBanner phase={phase} />

        {scene?.gmOutputText ? (
          <div className="rounded-box border border-base-300/70 bg-base-100/70 p-4">
            <div className="flex items-center gap-2 text-xs uppercase tracking-[0.2em] text-base-content/45">
              <span>Committed GM Output</span>
              {scene.gmOutputAuthor ? <span className="badge badge-ghost badge-xs">{scene.gmOutputAuthor}</span> : null}
            </div>
            <p className="mt-3 text-sm leading-7 text-base-content/85">{scene.gmOutputText}</p>
          </div>
        ) : null}

        {scene?.frameText ? (
          <div className="rounded-box border border-info/25 bg-info/10 p-4">
            <p className="text-xs uppercase tracking-[0.2em] text-info/70">Current Player Frame</p>
            <p className="mt-3 text-base leading-7 text-base-content/90">{scene.frameText}</p>
          </div>
        ) : null}
      </div>
    </section>
  );
}
