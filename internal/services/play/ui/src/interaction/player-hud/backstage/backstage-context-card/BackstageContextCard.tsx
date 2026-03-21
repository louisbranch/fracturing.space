import type { BackstageContextCardProps } from "./contract";
import { PlayerHUDStatusPill } from "../../shared/PlayerHUDStatusPill";

export function BackstageContextCard({
  sceneName,
  pausedPromptText,
  reason,
  status,
}: BackstageContextCardProps) {
  if (!sceneName && !pausedPromptText && !reason) {
    return null;
  }

  return (
    <section
      aria-label="Backstage context"
      className="border-b border-base-300/70 bg-base-300 px-3 py-3"
    >
      <div className="rounded-box border border-base-300/70 bg-base-100 px-3 py-3">
        <div className="flex flex-wrap items-start justify-between gap-2">
          <div className="flex flex-wrap items-center gap-2">
            {sceneName ? (
              <h2 className="text-base font-semibold text-base-content">{sceneName}</h2>
            ) : null}
            <span className="badge badge-sm badge-soft">Paused Scene</span>
          </div>
          <PlayerHUDStatusPill
            ariaLabel={`Backstage status: ${status.label}`}
            status={status}
          />
        </div>

        {pausedPromptText ? (
          <div className="mt-2">
            <div className="text-xs font-semibold uppercase tracking-wide text-base-content/55">
              Paused Prompt
            </div>
            <p className="mt-1 text-sm text-base-content/80">{pausedPromptText}</p>
          </div>
        ) : null}

        {reason ? (
          <div className="mt-2">
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
