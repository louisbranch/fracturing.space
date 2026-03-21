import type { OnStageSceneCardProps } from "./contract";
import { PlayerHUDStatusPill } from "../../shared/PlayerHUDStatusPill";

export function OnStageSceneCard({
  sceneName,
  sceneDescription,
  gmOutputText,
  frameText,
  actingCharacterNames,
  status,
}: OnStageSceneCardProps) {
  return (
    <section
      aria-label="On-stage scene context"
      className="border-b border-base-300/70 bg-base-100/80 px-3 py-3"
    >
      <div className="rounded-box border border-base-300/70 bg-base-100 px-3 py-3">
        <div className="flex flex-wrap items-start justify-between gap-2">
          <div className="flex flex-wrap items-center gap-2">
            <span className="badge badge-sm badge-soft">Active Scene</span>
            <h2 className="text-base font-semibold text-base-content">{sceneName}</h2>
          </div>
          <PlayerHUDStatusPill
            ariaLabel={`On-stage status: ${status.label}`}
            status={status}
          />
        </div>

        {sceneDescription ? (
          <p className="mt-2 text-sm text-base-content/75">{sceneDescription}</p>
        ) : null}

        {actingCharacterNames.length > 0 ? (
          <div className="mt-3 flex flex-wrap items-center gap-1.5">
            <div className="text-xs font-semibold uppercase tracking-wide text-base-content/55">
              Acting Now
            </div>
            <div className="flex flex-wrap gap-1.5">
              {actingCharacterNames.map((name) => (
                <span key={name} className="badge badge-outline badge-sm">
                  {name}
                </span>
              ))}
            </div>
          </div>
        ) : null}

        {gmOutputText ? (
          <div className="mt-3">
            <div className="text-xs font-semibold uppercase tracking-wide text-base-content/55">
              Latest GM Output
            </div>
            <p className="mt-1 text-sm text-base-content/80">{gmOutputText}</p>
          </div>
        ) : null}

        {frameText ? (
          <div className="mt-3">
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
