import { FoldVertical, UnfoldVertical } from "lucide-react";
import { OnStageCharacterAvatarStack } from "../on-stage-character-avatar-stack/OnStageCharacterAvatarStack";
import type { OnStageSceneCardProps } from "./contract";

const COLLAPSED_DESCRIPTION_CHAR_CAP = 150;

function truncateDescription(value: string): string {
  if (value.length <= COLLAPSED_DESCRIPTION_CHAR_CAP) {
    return value;
  }
  return `${value.slice(0, COLLAPSED_DESCRIPTION_CHAR_CAP)}...`;
}

export function OnStageSceneCard({
  sceneName,
  sceneDescription,
  sceneCharacters,
  expanded,
  onToggle,
  onCharacterInspect,
}: OnStageSceneCardProps) {
  return (
    <section
      aria-label="On-stage scene context"
      className="min-w-0 border-b border-base-300/70 bg-base-100/80 px-3 py-3"
    >
      <div className="min-w-0 rounded-box border border-base-300/70 bg-base-100 px-3 py-3">
        <div className="flex flex-wrap items-start justify-between gap-2">
          <div className="flex min-w-0 flex-wrap items-center gap-2">
            <h2 className="min-w-0 truncate text-base font-semibold text-base-content">{sceneName}</h2>
            <span className="badge badge-sm badge-soft">Active Scene</span>
          </div>
          {sceneCharacters.length > 0 ? (
            <OnStageCharacterAvatarStack
              characters={sceneCharacters}
              ariaLabel={`Scene characters: ${sceneCharacters.map((character) => character.name).join(", ")}`}
              onCharacterInspect={onCharacterInspect}
            />
          ) : null}
        </div>

        {sceneDescription ? (
          expanded ? (
            <div className="mt-3 flex items-start gap-2">
              <p className="min-w-0 flex-1 text-sm leading-6 text-base-content/75">{sceneDescription}</p>
              <button
                type="button"
                className="btn btn-ghost btn-xs shrink-0"
                onClick={onToggle}
                aria-expanded={expanded}
                aria-label="Collapse scene description"
              >
                <FoldVertical className="h-4 w-4" />
              </button>
            </div>
          ) : (
            <div className="mt-3 flex items-center gap-2">
              <p className="min-w-0 flex-1 truncate text-sm text-base-content/60">
                {truncateDescription(sceneDescription)}
              </p>
              <button
                type="button"
                className="btn btn-ghost btn-xs shrink-0"
                onClick={onToggle}
                aria-expanded={expanded}
                aria-label="Expand scene description"
              >
                <UnfoldVertical className="h-4 w-4" />
              </button>
            </div>
          )
        ) : null}
      </div>
    </section>
  );
}
