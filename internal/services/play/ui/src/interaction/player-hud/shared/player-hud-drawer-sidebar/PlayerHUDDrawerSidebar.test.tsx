import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { playerHUDFixtureCatalog } from "../fixtures";
import { PlayerHUDDrawerSidebar } from "./PlayerHUDDrawerSidebar";

describe("PlayerHUDDrawerSidebar", () => {
  it("starts with the Characters section collapsed", () => {
    render(
      <PlayerHUDDrawerSidebar
        navigation={playerHUDFixtureCatalog.onStage.campaignNavigation}
        onClose={() => {}}
      />,
    );

    const charactersButton = screen.getByRole("button", { name: "Characters" });
    expect(charactersButton).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByRole("button", { name: "Inspect Aria" })).not.toBeInTheDocument();
  });

  it("expands the character list in alphabetical order", async () => {
    const user = userEvent.setup();
    render(
      <PlayerHUDDrawerSidebar
        navigation={playerHUDFixtureCatalog.onStage.campaignNavigation}
        onClose={() => {}}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Characters" }));

    const buttons = screen.getAllByRole("button", { name: /Inspect / });
    expect(buttons.map((button) => button.getAttribute("aria-label"))).toEqual([
      "Inspect Aria",
      "Inspect Corin",
      "Inspect Mira",
      "Inspect Rowan",
      "Inspect Sable",
    ]);
  });

  it("emphasizes viewer-controlled characters", async () => {
    const user = userEvent.setup();
    render(
      <PlayerHUDDrawerSidebar
        navigation={playerHUDFixtureCatalog.onStage.campaignNavigation}
        onClose={() => {}}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Characters" }));

    const ariaButton = screen.getByRole("button", { name: "Inspect Aria" });
    expect(ariaButton).toHaveClass("border-primary/50");
    expect(within(ariaButton).getByText("You")).toBeInTheDocument();

    const sableButton = screen.getByRole("button", { name: "Inspect Sable" });
    expect(within(sableButton).queryByText("You")).not.toBeInTheDocument();
  });

  it("forwards character inspection and closes the drawer on selection", async () => {
    const user = userEvent.setup();
    const onCharacterInspect = vi.fn();
    const onClose = vi.fn();
    render(
      <PlayerHUDDrawerSidebar
        navigation={playerHUDFixtureCatalog.onStage.campaignNavigation}
        onCharacterInspect={onCharacterInspect}
        onClose={onClose}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Characters" }));
    await user.click(screen.getByRole("button", { name: "Inspect Sable" }));

    expect(onCharacterInspect).toHaveBeenCalledWith("p-ives", "char-sable");
    expect(onClose).toHaveBeenCalled();
  });

  it("renders Return to Campaign with the provided href", () => {
    render(
      <PlayerHUDDrawerSidebar
        navigation={playerHUDFixtureCatalog.onStage.campaignNavigation}
        onClose={() => {}}
      />,
    );

    expect(screen.getByRole("link", { name: "Return to Campaign" })).toHaveAttribute(
      "href",
      "/app/campaigns/camp-sealed-vault",
    );
  });
});
