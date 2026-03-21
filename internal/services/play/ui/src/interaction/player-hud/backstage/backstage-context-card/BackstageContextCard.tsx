import type { BackstageContextCardProps } from "./contract";

export function BackstageContextCard({
  sceneName,
  pausedPromptText,
  reason,
}: BackstageContextCardProps) {
  if (!sceneName && !pausedPromptText && !reason) {
    return null;
  }

  return (
    <section
      aria-label="Backstage context"
      className="border-b border-base-300/70 bg-base-100/80 px-4 py-4"
    >
      <div className="rounded-box border border-base-300/70 bg-base-100 px-4 py-3">
        <div className="flex flex-wrap items-center gap-2">
          <span className="badge badge-soft">Paused Scene</span>
          {sceneName ? (
            <span className="font-semibold text-base-content">{sceneName}</span>
          ) : null}
        </div>

        {pausedPromptText ? (
          <div className="mt-3">
            <div className="text-xs font-semibold uppercase tracking-wide text-base-content/55">
              Paused Prompt
            </div>
            <p className="mt-1 text-sm text-base-content/80">{pausedPromptText}</p>
          </div>
        ) : null}

        {reason ? (
          <div className="mt-3">
            <div className="text-xs font-semibold uppercase tracking-wide text-base-content/55">
              OOC Reason
            </div>
            <p className="mt-1 text-sm text-base-content/80">{reason}</p>
          </div>
        ) : null}
      </div>
    </section>
  );
}
