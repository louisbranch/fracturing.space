import { useMemo, useState } from "react";
import { BookOpen, SquareUser } from "lucide-react";
import { CharacterPortraitAvatar } from "../PlayerHUDCharacterInspector";
import type {
  PlayerHUDCharacterController,
} from "../contract";
import type { PlayerHUDCharacterReference } from "../character-inspection-contract";
import type { PlayerHUDDrawerSidebarProps } from "./contract";

type DrawerCharacterEntry = {
  participantId: string;
  participantName: string;
  isViewer: boolean;
  character: PlayerHUDCharacterReference;
};

const characterCollator = new Intl.Collator(undefined, {
  sensitivity: "base",
  usage: "sort",
});

function characterEntries(controllers: PlayerHUDCharacterController[]): DrawerCharacterEntry[] {
  const entries = controllers.flatMap((controller) =>
    controller.characters.map((character) => ({
      participantId: controller.participantId,
      participantName: controller.participantName,
      isViewer: controller.isViewer,
      character,
    })),
  );

  return entries.sort((left, right) => {
    const nameOrder = characterCollator.compare(left.character.name, right.character.name);
    if (nameOrder !== 0) {
      return nameOrder;
    }
    return characterCollator.compare(left.participantName, right.participantName);
  });
}

function characterAction(
  entry: DrawerCharacterEntry,
  onCharacterInspect: PlayerHUDDrawerSidebarProps["onCharacterInspect"],
  onClose: () => void,
) {
  onCharacterInspect?.(entry.participantId, entry.character.id);
  onClose();
}

export function PlayerHUDDrawerSidebar({
  navigation,
  onCharacterInspect,
  onClose,
}: PlayerHUDDrawerSidebarProps) {
  const [charactersExpanded, setCharactersExpanded] = useState(false);
  const entries = useMemo(
    () => characterEntries(navigation.characterControllers),
    [navigation.characterControllers],
  );

  return (
    <aside
      aria-label="Player HUD sidebar"
      className="min-h-full w-80 max-w-[85vw] bg-base-300 text-base-content shadow-2xl"
    >
      <div className="flex min-h-full flex-col gap-4 px-4 py-5">
        <section
          className={`collapse collapse-arrow border border-base-300/70 bg-base-100/70 ${
            charactersExpanded ? "collapse-open" : "collapse-close"
          }`}
        >
          <button
            type="button"
            aria-expanded={charactersExpanded}
            className="collapse-title flex cursor-pointer items-center gap-3 pr-10 text-left text-sm font-semibold"
            onClick={() => setCharactersExpanded((current) => !current)}
          >
            <SquareUser size={18} aria-hidden="true" />
            <span>Characters</span>
          </button>
          {charactersExpanded ? (
            <div className="collapse-content">
              {entries.length > 0 ? (
                <div className="flex flex-col gap-2">
                  {entries.map((entry) => {
                    const itemClassName = entry.isViewer
                      ? "border-primary/50 bg-primary/10"
                      : "border-base-300/70 bg-base-100/60 hover:border-primary/40";

                    return (
                      <button
                        key={`${entry.participantId}:${entry.character.id}`}
                        type="button"
                        aria-label={`Inspect ${entry.character.name}`}
                        className={`flex cursor-pointer items-center gap-3 rounded-box border px-3 py-2 text-left transition ${itemClassName}`}
                        onClick={() => characterAction(entry, onCharacterInspect, onClose)}
                      >
                        <CharacterPortraitAvatar
                          character={entry.character}
                          active={entry.isViewer}
                          sizeClassName="h-11 w-11"
                        />
                        <div className="min-w-0 flex-1">
                          <div className="truncate text-sm font-medium text-base-content">
                            {entry.character.name}
                          </div>
                        </div>
                        {entry.isViewer ? (
                          <span className="badge badge-primary badge-soft badge-xs">You</span>
                        ) : null}
                      </button>
                    );
                  })}
                </div>
              ) : (
                <p className="text-sm text-base-content/70">
                  No campaign characters are available yet.
                </p>
              )}
            </div>
          ) : null}
        </section>

        <div className="divider my-0" />

        <a
          href={navigation.returnHref}
          className="btn btn-ghost cursor-pointer justify-start gap-3 px-3"
          onClick={onClose}
        >
          <BookOpen size={18} aria-hidden="true" />
          <span>Return to Campaign</span>
        </a>
      </div>
    </aside>
  );
}
