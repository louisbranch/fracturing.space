import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { OnStagePanelPreview } from "./OnStagePanelPreview";
import { onStagePanelFixtures } from "./fixtures";

describe("OnStagePanelPreview", () => {
  it("opens the clicked character from a multi-character on-stage slot", async () => {
    const user = userEvent.setup();

    render(
      <OnStagePanelPreview
        initialState={onStagePanelFixtures.multiCharacterOwner}
      />,
    );

    await user.click(screen.getAllByRole("button", { name: "Inspect Aria" })[1]);

    const dialog = screen.getByRole("dialog");

    expect(within(dialog).getByRole("heading", { name: "Rhea" })).toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: "Aria" })).toBeInTheDocument();

    await user.click(
      within(dialog).getByRole("button", { name: "Character Sheet" }),
    );
    expect(within(dialog).getByText("Damage & Health")).toBeInTheDocument();
  });
});
