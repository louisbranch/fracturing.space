import { Component, useEffect, useId, useRef, useState } from "react";
import { X } from "lucide-react";
import { CharacterCard } from "../../../systems/daggerheart/character-card/CharacterCard";
import { CharacterSheet } from "../../../systems/daggerheart/character-sheet/CharacterSheet";
import type {
  PlayerHUDCharacterInspectionCatalog,
  PlayerHUDCharacterReference,
} from "./character-inspection-contract";

type InspectorParticipant = {
  name: string;
  characters: PlayerHUDCharacterReference[];
  isViewer?: boolean;
};

type InspectorState = {
  participantName: string;
  characters: PlayerHUDCharacterReference[];
  activeCharacterId?: string;
  isViewer: boolean;
};

type CharacterInspectorRenderBoundaryProps = {
  boundaryKey: string;
  viewMode: "card" | "sheet";
  characterName: string;
  children: React.ReactNode;
};

type CharacterInspectorRenderBoundaryState = {
  error: Error | null;
};

export type PlayerHUDCharacterInspectorDialogProps = {
  isOpen: boolean;
  participantName: string;
  characters: PlayerHUDCharacterReference[];
  activeCharacterId?: string;
  isViewer?: boolean;
  characterInspectionCatalog: PlayerHUDCharacterInspectionCatalog;
  onCharacterChange: (characterId: string) => void;
  onClose: () => void;
};

export function usePlayerHUDCharacterInspector() {
  const [state, setState] = useState<InspectorState | null>(null);

  function openForParticipant(participant: InspectorParticipant) {
    setState({
      participantName: participant.name,
      characters: participant.characters,
      activeCharacterId: participant.characters[0]?.id,
      isViewer: Boolean(participant.isViewer),
    });
  }

  function openForCharacter(
    participant: InspectorParticipant,
    characterId: string,
  ) {
    const hasSelectedCharacter = participant.characters.some(
      (character) => character.id === characterId,
    );

    setState({
      participantName: participant.name,
      characters: participant.characters,
      activeCharacterId: hasSelectedCharacter
        ? characterId
        : participant.characters[0]?.id,
      isViewer: Boolean(participant.isViewer),
    });
  }

  function close() {
    setState(null);
  }

  function setActiveCharacter(characterId: string) {
    setState((current) =>
      current
        ? {
            ...current,
            activeCharacterId: characterId,
          }
        : current,
    );
  }

  return {
    inspector: state,
    openForParticipant,
    openForCharacter,
    setActiveCharacter,
    close,
  };
}

export function PlayerHUDCharacterInspectorDialog({
  isOpen,
  participantName,
  characters,
  activeCharacterId,
  isViewer = false,
  characterInspectionCatalog,
  onCharacterChange,
  onClose,
}: PlayerHUDCharacterInspectorDialogProps) {
  const dialogRef = useRef<HTMLDialogElement | null>(null);
  const wasOpenRef = useRef(false);
  const [viewMode, setViewMode] = useState<"card" | "sheet">("card");
  const titleID = useId();
  const activeCharacter = characters.find(
    (character) => character.id === activeCharacterId,
  );
  const activeInspection = activeCharacterId
    ? characterInspectionCatalog[activeCharacterId]
    : undefined;
  const activeCard = activeInspection?.card;
  const activeSheet = activeInspection?.sheet;
  const hasCharacters = characters.length > 0;
  const hasActiveInspection = Boolean(activeCharacter && activeInspection);

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) {
      return;
    }

    if (isOpen && !dialog.open) {
      if (typeof dialog.showModal === "function") {
        dialog.showModal();
      } else {
        dialog.setAttribute("open", "");
      }
    }

    if (!isOpen && dialog.open) {
      if (typeof dialog.close === "function") {
        dialog.close();
      } else {
        dialog.removeAttribute("open");
      }
    }
  }, [isOpen]);

  useEffect(() => {
    if (isOpen && !wasOpenRef.current) {
      setViewMode("card");
    }
    wasOpenRef.current = isOpen;
  }, [isOpen]);

  return (
    <dialog
      ref={dialogRef}
      aria-labelledby={titleID}
      className="modal"
      onClose={onClose}
    >
      <div className="modal-box flex max-h-[90vh] max-w-5xl flex-col gap-4 overflow-hidden p-0">
        <header className="flex items-start justify-between gap-4 border-b border-base-300/70 px-5 py-4">
          <div className="flex items-center gap-2">
            <h2 id={titleID} className="text-lg font-semibold text-base-content">
              {participantName}
            </h2>
            {isViewer ? (
              <span className="badge badge-primary badge-soft badge-sm">You</span>
            ) : null}
          </div>

          <button
            type="button"
            aria-label="Close character inspector"
            className="btn btn-ghost btn-sm btn-square"
            onClick={onClose}
          >
            <X size={18} aria-hidden="true" />
          </button>
        </header>

        {characters.length > 1 ? (
          <section
            aria-label={`${participantName} characters`}
            className="flex flex-wrap gap-2 px-5 pt-1"
          >
            {characters.map((character) => {
              const isActive = character.id === activeCharacterId;

              return (
                <button
                  key={character.id}
                  type="button"
                  className={`flex items-center gap-2 rounded-box border px-2 py-1.5 text-left transition ${
                    isActive
                      ? "border-primary bg-primary/10 ring-2 ring-primary/40"
                      : "border-base-300/70 bg-base-200/30 hover:border-primary/50"
                  }`}
                  onClick={() => onCharacterChange(character.id)}
                >
                  <CharacterPortraitAvatar
                    character={character}
                    active={isActive}
                    sizeClassName="h-10 w-10"
                  />
                  <span className="text-sm font-medium text-base-content">
                    {character.name}
                  </span>
                </button>
              );
            })}
          </section>
        ) : null}

        <div className="min-h-0 flex-1 overflow-y-auto px-5 pb-1">
          {hasActiveInspection ? (
            <CharacterInspectorRenderBoundary
              boundaryKey={`${viewMode}:${activeCharacterId ?? ""}`}
              viewMode={viewMode}
              characterName={activeCharacter?.name ?? participantName}
            >
              {viewMode === "card" ? (
                <CharacterCard
                  character={activeCard!}
                  variant="basic"
                />
              ) : (
                <CharacterSheet character={activeSheet!} />
              )}
            </CharacterInspectorRenderBoundary>
          ) : hasCharacters ? (
            <section className="rounded-box border border-warning/40 bg-warning/10 px-4 py-6 text-sm text-base-content/75">
              Character details are not available for the selected portrait yet.
            </section>
          ) : (
            <section className="rounded-box border border-base-300/70 bg-base-200/30 px-4 py-6 text-sm text-base-content/75">
              No character sheet is available for this participant yet.
            </section>
          )}
        </div>

        <footer className="flex flex-wrap items-center justify-center gap-2 border-t border-base-300/70 px-5 py-4">
          {hasActiveInspection ? (
            viewMode === "card" ? (
              <button
                type="button"
                className="btn btn-primary"
                onClick={() => setViewMode("sheet")}
              >
                Character Sheet
              </button>
            ) : (
              <button
                type="button"
                className="btn btn-outline"
                onClick={() => setViewMode("card")}
              >
                Back
              </button>
            )
          ) : (
            <button type="button" className="btn btn-disabled" disabled>
              Character Sheet unavailable
            </button>
          )}
        </footer>
      </div>
      <form method="dialog" className="modal-backdrop">
        <button type="submit" onClick={onClose}>
          close
        </button>
      </form>
    </dialog>
  );
}

class CharacterInspectorRenderBoundary extends Component<
  CharacterInspectorRenderBoundaryProps,
  CharacterInspectorRenderBoundaryState
> {
  state: CharacterInspectorRenderBoundaryState = {
    error: null,
  };

  static getDerivedStateFromError(error: Error): CharacterInspectorRenderBoundaryState {
    return { error };
  }

  componentDidCatch(error: Error) {
    console.error("[play character inspector] render failed", {
      boundaryKey: this.props.boundaryKey,
      viewMode: this.props.viewMode,
      characterName: this.props.characterName,
      error: error.message,
    });
  }

  componentDidUpdate(prevProps: CharacterInspectorRenderBoundaryProps) {
    if (prevProps.boundaryKey !== this.props.boundaryKey && this.state.error) {
      this.setState({ error: null });
    }
  }

  render() {
    if (this.state.error) {
      return (
        <section className="rounded-box border border-error/40 bg-error/10 px-4 py-6 text-sm text-base-content/75">
          <p className="font-medium text-base-content">Character details could not be rendered.</p>
          <p className="mt-2">
            The {this.props.viewMode === "sheet" ? "sheet" : "card"} view for {this.props.characterName} failed to load.
          </p>
          <p className="mt-1 text-base-content/60">
            Try switching characters or returning to the other view.
          </p>
        </section>
      );
    }

    return this.props.children;
  }
}

export function CharacterPortraitAvatar(input: {
  character: PlayerHUDCharacterReference;
  active?: boolean;
  sizeClassName?: string;
}) {
  const initials = input.character.name
    .split(/\s+/)
    .filter(Boolean)
    .map((segment) => segment[0]?.toUpperCase() ?? "")
    .slice(0, 2)
    .join("");

  return (
    <div
      className={`overflow-hidden rounded-full border bg-base-300 text-base-content shadow-sm ${
        input.active ? "border-primary" : "border-base-100"
      } ${input.sizeClassName ?? "h-8 w-8"}`}
    >
      {input.character.avatarUrl ? (
        <img
          src={input.character.avatarUrl}
          alt=""
          className="h-full w-full object-cover"
        />
      ) : (
        <div className="flex h-full w-full items-center justify-center text-xs font-semibold">
          {initials || "?"}
        </div>
      )}
    </div>
  );
}
