import { CharacterCard } from "../../../systems/daggerheart/character-card/CharacterCard";
import { CharacterSheet } from "../../../systems/daggerheart/character-sheet/CharacterSheet";
import type { CharacterReferenceRailProps } from "./contract";

// CharacterReferenceRail reuses the isolated Daggerheart character surfaces as
// read-only reference tools inside the interaction shell.
export function CharacterReferenceRail({
  characters,
  activeCharacterIds,
  selectedCharacterId,
  selectedSheet,
  onSelectCharacter,
}: CharacterReferenceRailProps) {
  return (
    <section className="preview-panel" aria-label="Character reference rail">
      <div className="preview-panel-body gap-4">
        <div>
          <span className="preview-kicker">Character Reference</span>
          <h2 className="font-display text-2xl text-base-content">Scene Roster</h2>
        </div>

        <div className="grid gap-4 lg:grid-cols-2">
          {characters.map((character) => {
            const active = activeCharacterIds.includes(character.id);
            const selected = selectedCharacterId === character.id;
            return (
              <div key={character.id} className="space-y-2">
                <div className="flex flex-wrap items-center gap-2">
                  {active ? <span className="badge badge-success badge-soft">Active</span> : null}
                  {selected ? <span className="badge badge-warning badge-soft">Selected</span> : null}
                </div>
                <button
                  className="block w-full text-left"
                  onClick={() => onSelectCharacter?.(character.id)}
                  type="button"
                >
                  <CharacterCard character={character} variant="basic" />
                </button>
              </div>
            );
          })}
        </div>

        {selectedSheet ? (
          <div className="rounded-box border border-base-300/70 bg-base-200/35 p-3">
            <p className="mb-3 text-xs uppercase tracking-[0.18em] text-base-content/45">Selected Character Sheet</p>
            <CharacterSheet character={selectedSheet} />
          </div>
        ) : null}
      </div>
    </section>
  );
}
