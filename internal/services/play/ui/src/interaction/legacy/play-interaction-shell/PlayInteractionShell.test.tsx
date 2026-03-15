import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { PlayInteractionShell } from "./PlayInteractionShell";
import { playInteractionShellFixtures } from "./fixtures";

describe("PlayInteractionShell", () => {
  it("assembles the isolated interaction slices into one reviewable shell", () => {
    render(<PlayInteractionShell {...playInteractionShellFixtures.playersOpen} />);

    expect(screen.getByText("Cliffside Rescue")).toBeInTheDocument();
    expect(screen.getByLabelText("Active scene frame")).toBeInTheDocument();
    expect(screen.getByLabelText("Acting set")).toBeInTheDocument();
    expect(screen.getByLabelText("Player slot board")).toBeInTheDocument();
    expect(screen.getByLabelText("Character reference rail")).toBeInTheDocument();
    expect(screen.getByLabelText("Session chat")).toBeInTheDocument();
  });
});
