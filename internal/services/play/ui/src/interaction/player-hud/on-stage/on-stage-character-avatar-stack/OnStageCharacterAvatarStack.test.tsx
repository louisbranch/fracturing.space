import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { onStageFixtureCatalog } from "../shared/fixtures";
import { OnStageCharacterAvatarStack } from "./OnStageCharacterAvatarStack";

describe("OnStageCharacterAvatarStack", () => {
  it("renders up to three character avatars without an ellipsis", () => {
    const viewerSlot = onStageFixtureCatalog.viewerPosted.slots[0];
    if (!viewerSlot) {
      throw new Error("expected viewer slot fixture");
    }

    const { container } = render(<OnStageCharacterAvatarStack characters={viewerSlot.characters} />);

    expect(screen.getByLabelText("Characters: Aria")).toBeInTheDocument();
    expect(container.querySelectorAll("img")).toHaveLength(1);
    expect(screen.queryByText("...")).not.toBeInTheDocument();
  });

  it("collapses four or more characters to two avatars and an ellipsis marker", () => {
    const multiCharacterSlot = onStageFixtureCatalog.multiCharacterOwner.slots[0];
    if (!multiCharacterSlot) {
      throw new Error("expected multi-character fixture");
    }

    const { container } = render(<OnStageCharacterAvatarStack characters={multiCharacterSlot.characters} />);

    expect(screen.getByLabelText("Characters: Aria, Sable, Mira, Rowan")).toBeInTheDocument();
    expect(container.querySelectorAll("img")).toHaveLength(2);
    expect(screen.getByText("...")).toBeInTheDocument();
  });

  it("emits the clicked character id when avatars are interactive", async () => {
    const user = userEvent.setup();
    const onCharacterInspect = vi.fn();
    const multiCharacterSlot = onStageFixtureCatalog.multiCharacterOwner.slots[0];
    if (!multiCharacterSlot) {
      throw new Error("expected multi-character fixture");
    }

    render(
      <OnStageCharacterAvatarStack
        characters={multiCharacterSlot.characters}
        onCharacterInspect={onCharacterInspect}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Inspect Aria" }));
    expect(onCharacterInspect).toHaveBeenCalledWith("char-aria");
    expect(screen.getByText("...")).toBeInTheDocument();
  });
});
