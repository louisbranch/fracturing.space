import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { PlayerHUDShell } from "./PlayerHUDShell";
import { playerHUDShellFixtures } from "./fixtures";

describe("PlayerHUDShell", () => {
  it("assembles the fixed player HUD slices into one viewport", () => {
    render(<PlayerHUDShell state={playerHUDShellFixtures.playerTurn} />);

    expect(screen.getByLabelText("Player HUD shell")).toBeInTheDocument();
    expect(screen.getByLabelText("Player HUD header")).toBeInTheDocument();
    expect(screen.getByLabelText("Player stage viewport")).toBeInTheDocument();
    expect(screen.getByLabelText("Player composer")).toBeInTheDocument();
    expect(screen.getByText("Cliffside Rescue")).toBeInTheDocument();
  });
});
