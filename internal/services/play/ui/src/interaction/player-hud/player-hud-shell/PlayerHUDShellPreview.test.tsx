import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { playerHUDShellFixtures } from "./fixtures";
import { PlayerHUDShellPreview } from "./PlayerHUDShellPreview";

describe("PlayerHUDShellPreview", () => {
  it("opens character inspection from the participant rail and shows the empty state for the GM portrait", async () => {
    const user = userEvent.setup();

    render(
      <PlayerHUDShellPreview initialState={playerHUDShellFixtures.onStage} />,
    );

    await user.click(screen.getByRole("button", { name: "Inspect Rhea" }));
    const dialog = screen.getByRole("dialog");
    expect(within(dialog).getByRole("heading", { name: "Rhea" })).toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: "Aria" })).toBeInTheDocument();

    await user.click(
      within(dialog).getByRole("button", { name: "Close character inspector" }),
    );

    await user.click(screen.getByRole("button", { name: "Inspect Guide" }));
    const emptyStateDialog = screen.getByRole("dialog");
    expect(
      within(emptyStateDialog).getByText(
        "No character sheet is available for this participant yet.",
      ),
    ).toBeInTheDocument();
  });
});
