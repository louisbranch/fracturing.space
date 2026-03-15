import type { AITurnStatusBannerProps } from "./contract";

const statusPresentation = {
  idle: {
    badge: "Idle",
    className: "border-base-300/70 bg-base-200/60",
    description: "No AI GM work is active for this interaction state.",
  },
  queued: {
    badge: "Queued",
    className: "border-info/30 bg-info/10",
    description: "The AI GM turn has been queued and is waiting to run.",
  },
  running: {
    badge: "Running",
    className: "border-warning/30 bg-warning/10",
    description: "The AI GM is currently executing against the campaign-scoped tool set.",
  },
  failed: {
    badge: "Failed",
    className: "border-error/30 bg-error/10",
    description: "The last authoritative AI GM turn failed before completion.",
  },
} as const;

// AITurnStatusBanner makes the cross-service AI lifecycle visible as one
// compact, reusable slice.
export function AITurnStatusBanner({ aiTurn, onRetry }: AITurnStatusBannerProps) {
  const presentation = statusPresentation[aiTurn.status];

  return (
    <section className={`rounded-box border p-4 ${presentation.className}`} aria-label="AI turn status">
      <div className="flex flex-wrap items-center gap-2">
        <span className="badge badge-outline">{presentation.badge}</span>
        {aiTurn.ownerName ? <span className="badge badge-ghost badge-sm">{aiTurn.ownerName}</span> : null}
        {aiTurn.sourceLabel ? <span className="badge badge-soft badge-sm">{aiTurn.sourceLabel}</span> : null}
      </div>

      <p className="mt-3 text-sm leading-6 text-base-content/85">{presentation.description}</p>

      {aiTurn.lastError ? (
        <p className="mt-3 text-sm leading-6 text-base-content/75">{aiTurn.lastError}</p>
      ) : null}

      {aiTurn.status === "failed" && aiTurn.canRetry ? (
        <div className="mt-4">
          <button className="btn btn-error btn-sm" onClick={onRetry} type="button">
            Retry AI Turn
          </button>
        </div>
      ) : null}
    </section>
  );
}
