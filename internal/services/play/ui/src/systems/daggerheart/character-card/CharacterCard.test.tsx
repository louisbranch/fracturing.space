import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { CharacterCard } from "./CharacterCard";
import { characterCardFixtures } from "./fixtures";

describe("CharacterCard", () => {
  it("renders portrait-only mode with accessible portrait text", () => {
    render(<CharacterCard character={characterCardFixtures.full} variant="portrait" />);

    expect(
      screen.getByRole("img", {
        name: /portrait of mira/i,
      }),
    ).toBeInTheDocument();
    expect(screen.getByText("Mira", { selector: ".sr-only" })).toBeInTheDocument();
  });

  it("renders basic mode with quick identity details", () => {
    render(<CharacterCard character={characterCardFixtures.full} variant="basic" />);

    const card = screen.getByRole("heading", { level: 2, name: "Mira" }).closest("article");
    const basicTraits = screen.getByLabelText("Character traits");
    expect(card).not.toBeNull();
    expect(within(card as HTMLElement).getByText("(she/her)")).toBeInTheDocument();
    expect(within(card as HTMLElement).getByText("Rogue / Nightwalker")).toBeInTheDocument();
    expect(within(card as HTMLElement).getByText("Human, Slyborne")).toBeInTheDocument();
    expect(within(card as HTMLElement).getByText("3/5")).toBeInTheDocument();
    expect(within(basicTraits).queryByText("Traits")).not.toBeInTheDocument();
    expect(within(basicTraits).getByText("AGI 2")).toBeInTheDocument();
    // Invariant: the basic card must stay compact and not introduce the full Daggerheart detail summary.
    expect(within(card as HTMLElement).queryByLabelText("Daggerheart full info")).not.toBeInTheDocument();
  });

  it("falls back to a placeholder portrait when the image source is missing", () => {
    render(<CharacterCard character={characterCardFixtures.partial} variant="basic" />);

    expect(
      screen.getByRole("img", {
        name: /portrait placeholder for zara/i,
      }),
    ).toBeInTheDocument();
    expect(screen.getByText("Zara")).toBeInTheDocument();
  });
});
