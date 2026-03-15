import type { StageViewportProps } from "./contract";

// StageViewport is the central player-facing slot where future narration and
// prompts will render, while keeping any overflow inside the stage itself.
export function StageViewport({ stage }: StageViewportProps) {
  const hasContent = stage.content.length > 0;

  return (
    <section className="preview-panel hud-stage" aria-label="Player stage viewport">
      <header className="hud-stage-header">
        <h2 className="font-display text-3xl text-base-content">Scene: {stage.title}</h2>
      </header>

      <div className="hud-stage-scroll overflow-y-auto">
        {hasContent ? (
          <div className="space-y-4">
            {stage.content.map((paragraph, index) => (
              <p key={`stage-paragraph-${index}`} className="max-w-4xl text-sm leading-7 text-base-content/82">
                {paragraph}
              </p>
            ))}
          </div>
        ) : (
          <div className="hud-stage-empty">
            <p className="max-w-xl text-sm leading-7 text-base-content/65">{stage.emptyMessage}</p>
          </div>
        )}
      </div>
    </section>
  );
}
