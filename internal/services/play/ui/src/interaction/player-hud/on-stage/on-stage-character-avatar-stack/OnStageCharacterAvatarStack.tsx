import type { OnStageCharacterAvatarStackProps } from "./contract";

function initials(name: string): string {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .map((segment) => segment[0]?.toUpperCase() ?? "")
    .slice(0, 2)
    .join("");
}

export function OnStageCharacterAvatarStack({
  characters,
  ariaLabel,
}: OnStageCharacterAvatarStackProps) {
  if (characters.length === 0) {
    return null;
  }

  const visibleCharacters = characters.length > 3 ? characters.slice(0, 2) : characters.slice(0, 3);
  const accessibleLabel = ariaLabel ?? `Characters: ${characters.map((character) => character.name).join(", ")}`;

  return (
    <div aria-label={accessibleLabel} className="flex items-center -space-x-2.5">
      {visibleCharacters.map((character) => (
        <div
          key={character.id}
          className="relative h-8 w-8 overflow-hidden rounded-full border-2 border-base-100 bg-base-300 text-base-content shadow-sm"
        >
          {character.avatarUrl ? (
            <img src={character.avatarUrl} alt="" className="h-full w-full object-cover" />
          ) : (
            <div className="flex h-full w-full items-center justify-center text-xs font-semibold">
              {initials(character.name) || "?"}
            </div>
          )}
        </div>
      ))}
      {characters.length > 3 ? (
        <div className="relative flex h-8 w-8 items-center justify-center rounded-full border-2 border-base-100 bg-base-200 text-xs font-semibold text-base-content shadow-sm">
          <span aria-hidden="true">...</span>
        </div>
      ) : null}
    </div>
  );
}
