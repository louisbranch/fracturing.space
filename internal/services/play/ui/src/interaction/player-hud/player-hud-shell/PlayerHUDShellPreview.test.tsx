import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { playerHUDShellFixtures } from "./fixtures";
import { PlayerHUDShellPreview } from "./PlayerHUDShellPreview";

describe("PlayerHUDShellPreview", () => {
  it("opens the drawer from the navbar and lists the campaign characters", async () => {
    const user = userEvent.setup();

    render(
      <PlayerHUDShellPreview initialState={playerHUDShellFixtures.onStage} />,
    );

    await user.click(screen.getByRole("button", { name: "Open campaign sidebar" }));
    const sidebar = screen.getByLabelText("Player HUD sidebar");
    expect(sidebar).toBeInTheDocument();

    await user.click(within(sidebar).getByRole("button", { name: "Characters" }));
    expect(within(sidebar).getByRole("button", { name: "Inspect Sable" })).toBeInTheDocument();
  });

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

  it("opens the same character inspector from the drawer with the selected character active", async () => {
    const user = userEvent.setup();

    render(
      <PlayerHUDShellPreview initialState={playerHUDShellFixtures.onStage} />,
    );

    await user.click(screen.getByRole("button", { name: "Open campaign sidebar" }));
    const sidebar = screen.getByLabelText("Player HUD sidebar");
    await user.click(within(sidebar).getByRole("button", { name: "Characters" }));
    await user.click(within(sidebar).getByRole("button", { name: "Inspect Sable" }));

    const dialog = screen.getByRole("dialog");
    expect(within(dialog).getByRole("heading", { name: "Ives" })).toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: "Sable" })).toBeInTheDocument();
    expect(screen.getByLabelText("Player HUD sidebar toggle")).not.toBeChecked();
  });
});
