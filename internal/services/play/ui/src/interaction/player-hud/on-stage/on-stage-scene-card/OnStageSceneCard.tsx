import type { OnStageSceneCardProps } from "./contract";

export function OnStageSceneCard({
  sceneName,
  sceneDescription,
  gmOutputText,
  frameText,
  actingCharacterNames,
  statusLabel,
  statusClassName,
  statusTooltip,
}: OnStageSceneCardProps) {
  return (
    <section
      aria-label="On-stage scene context"
      className="border-b border-base-300/70 bg-base-100/80 px-4 py-4"
    >
      <div className="rounded-box border border-base-300/70 bg-base-100 px-4 py-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="flex flex-wrap items-center gap-2">
            <span className="badge badge-soft">Active Scene</span>
            <h2 className="text-lg font-semibold text-base-content">{sceneName}</h2>
          </div>
          <span
            aria-label={`On-stage status: ${statusLabel}`}
            className="tooltip tooltip-bottom shrink-0"
            data-tip={statusTooltip}
          >
            <span className={`badge ${statusClassName}`} tabIndex={0}>
              {statusLabel}
            </span>
          </span>
        </div>

        {sceneDescription ? (
          <p className="mt-3 text-sm text-base-content/75">{sceneDescription}</p>
        ) : null}

        {actingCharacterNames.length > 0 ? (
          <div className="mt-4">
            <div className="text-xs font-semibold uppercase tracking-wide text-base-content/55">
              Acting Now
            </div>
            <div className="mt-2 flex flex-wrap gap-2">
              {actingCharacterNames.map((name) => (
                <span key={name} className="badge badge-outline badge-sm">
                  {name}
                </span>
              ))}
            </div>
          </div>
        ) : null}

        {gmOutputText ? (
          <div className="mt-4">
            <div className="text-xs font-semibold uppercase tracking-wide text-base-content/55">
              Latest GM Output
            </div>
            <p className="mt-1 text-sm text-base-content/80">{gmOutputText}</p>
          </div>
        ) : null}

        {frameText ? (
          <div className="mt-4">
            <div className="text-xs font-semibold uppercase tracking-wide text-base-content/55">
              Current Frame
            </div>
            <p className="mt-1 text-sm text-base-content/80">{frameText}</p>
          </div>
        ) : null}
      </div>
    </section>
  );
}
