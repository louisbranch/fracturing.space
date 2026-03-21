import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { PlayerComposerPreview } from "./PlayerComposerPreview";
import { playerComposerFixtures } from "./fixtures";

describe("PlayerComposer", () => {
  it("keeps separate drafts when switching between composer modes", async () => {
    const user = userEvent.setup();

    render(<PlayerComposerPreview initialState={playerComposerFixtures.gmTurn} />);

    const scratchDraft = screen.getByLabelText("Scratch Pad draft");
    expect(scratchDraft).toHaveValue(playerComposerFixtures.gmTurn.drafts.scratch);

    await user.click(screen.getByRole("tab", { name: "Chat" }));
    await user.clear(screen.getByLabelText("Chat draft"));
    await user.type(screen.getByLabelText("Chat draft"), "Scout is back on the line.");

    await user.click(screen.getByRole("tab", { name: "Scratch Pad" }));

    expect(screen.getByLabelText("Scratch Pad draft")).toHaveValue(playerComposerFixtures.gmTurn.drafts.scratch);
    await user.click(screen.getByRole("tab", { name: "Chat" }));
    expect(screen.getByLabelText("Chat draft")).toHaveValue("Scout is back on the line.");
  });

  it("toggles yield controls when the scene composer is active", async () => {
    const user = userEvent.setup();

    render(<PlayerComposerPreview initialState={playerComposerFixtures.playerTurn} />);

    expect(screen.getByRole("button", { name: "Yield" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Yield" }));
    expect(screen.getByRole("button", { name: "Unyield" })).toBeInTheDocument();
  });

  it("switches OOC controls between pause and resume", async () => {
    const user = userEvent.setup();

    render(<PlayerComposerPreview initialState={playerComposerFixtures.gmTurn} />);

    await user.click(screen.getByRole("tab", { name: "Out Of Character" }));
    expect(screen.getByRole("button", { name: "Pause" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Pause" }));

    expect(screen.getByRole("button", { name: "Resume" })).toBeInTheDocument();
    expect(screen.getByLabelText("Out Of Character draft")).toBeEnabled();
  });

  it("preserves the active mode when minimizing and expanding", async () => {
    const user = userEvent.setup();

    render(<PlayerComposerPreview initialState={playerComposerFixtures.playerTurn} />);

    await user.click(screen.getByRole("tab", { name: "Chat" }));
    await user.click(screen.getByRole("button", { name: "Minimize composer" }));
    expect(screen.getByRole("tablist", { name: "Player composer modes" })).toBeInTheDocument();
    expect(screen.queryByRole("tabpanel")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Expand composer" }));
    expect(screen.getByRole("tab", { name: "Chat" })).toHaveAttribute("aria-selected", "true");
  });
});
