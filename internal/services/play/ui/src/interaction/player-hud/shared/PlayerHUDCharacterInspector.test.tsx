const inspectorRenderFailureState = vi.hoisted(() => ({ shouldThrowSheet: false }));

vi.mock("../../../systems/daggerheart/character-sheet/CharacterSheet", async () => {
  const actual = await vi.importActual<typeof import("../../../systems/daggerheart/character-sheet/CharacterSheet")>(
    "../../../systems/daggerheart/character-sheet/CharacterSheet",
  );

  return {
    ...actual,
    CharacterSheet: (props: import("../../../systems/daggerheart/character-sheet/contract").CharacterSheetProps) => {
      if (inspectorRenderFailureState.shouldThrowSheet) {
        throw new Error("sheet exploded");
      }
      return actual.CharacterSheet(props);
    },
  };
});

import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  playerHUDCharacterCatalog,
  playerHUDCharacterInspectionCatalog,
} from "./character-inspection-fixtures";
import { PlayerHUDCharacterInspectorDialog } from "./PlayerHUDCharacterInspector";

afterEach(() => {
  inspectorRenderFailureState.shouldThrowSheet = false;
});

describe("PlayerHUDCharacterInspectorDialog", () => {
  it("opens on the full card, toggles to the full sheet, and preserves view while switching characters", async () => {
    const user = userEvent.setup();

    render(
      <PlayerHUDCharacterInspectorDialog
        isOpen
        participantName="Rhea"
        characters={[
          playerHUDCharacterCatalog.aria,
          playerHUDCharacterCatalog.mira,
        ]}
        activeCharacterId={playerHUDCharacterCatalog.aria.id}
        isViewer
        characterInspectionCatalog={playerHUDCharacterInspectionCatalog}
        onCharacterChange={vi.fn()}
        onClose={vi.fn()}
      />,
    );

    const dialog = screen.getByRole("dialog");

    expect(within(dialog).getByRole("heading", { name: "Rhea" })).toBeInTheDocument();
    expect(within(dialog).getByText("You")).toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: "Aria" })).toBeInTheDocument();
    expect(
      within(dialog).getByRole("button", { name: "Character Sheet" }),
    ).toBeInTheDocument();

    await user.click(
      within(dialog).getByRole("button", { name: "Character Sheet" }),
    );

    expect(
      within(dialog).getByRole("button", { name: "Back" }),
    ).toBeInTheDocument();
    expect(within(dialog).getByText("Damage & Health")).toBeInTheDocument();
  });

  it("switches the active character from the portrait strip", async () => {
    const user = userEvent.setup();
    const onCharacterChange = vi.fn();

    render(
      <PlayerHUDCharacterInspectorDialog
        isOpen
        participantName="Rhea"
        characters={[
          playerHUDCharacterCatalog.aria,
          playerHUDCharacterCatalog.mira,
        ]}
        activeCharacterId={playerHUDCharacterCatalog.aria.id}
        isViewer
        characterInspectionCatalog={playerHUDCharacterInspectionCatalog}
        onCharacterChange={onCharacterChange}
        onClose={vi.fn()}
      />,
    );

    const dialog = screen.getByRole("dialog");

    await user.click(within(dialog).getByRole("button", { name: "Mira" }));
    expect(onCharacterChange).toHaveBeenCalledWith(playerHUDCharacterCatalog.mira.id);
  });

  it("renders an empty-state modal when the participant has no characters", () => {
    render(
      <PlayerHUDCharacterInspectorDialog
        isOpen
        participantName="Guide"
        characters={[]}
        characterInspectionCatalog={playerHUDCharacterInspectionCatalog}
        onCharacterChange={vi.fn()}
        onClose={vi.fn()}
      />,
    );

    expect(screen.queryByRole("button", { name: "Close" })).not.toBeInTheDocument();
    expect(
      screen.getByText("No character sheet is available for this participant yet."),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Character Sheet unavailable" }),
    ).toBeDisabled();
  });

  it("contains sheet render failures inside the modal and allows recovery", async () => {
    const user = userEvent.setup();
    inspectorRenderFailureState.shouldThrowSheet = true;

    render(
      <PlayerHUDCharacterInspectorDialog
        isOpen
        participantName="Rhea"
        characters={[playerHUDCharacterCatalog.aria]}
        activeCharacterId={playerHUDCharacterCatalog.aria.id}
        characterInspectionCatalog={playerHUDCharacterInspectionCatalog}
        onCharacterChange={vi.fn()}
        onClose={vi.fn()}
      />,
    );

    const dialog = screen.getByRole("dialog");

    await user.click(within(dialog).getByRole("button", { name: "Character Sheet" }));

    expect(
      within(dialog).getByText("Character details could not be rendered."),
    ).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Back" })).toBeInTheDocument();

    inspectorRenderFailureState.shouldThrowSheet = false;
    await user.click(within(dialog).getByRole("button", { name: "Back" }));

    expect(within(dialog).getByRole("heading", { name: "Aria" })).toBeInTheDocument();
  });
});
