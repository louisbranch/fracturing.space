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
    expect(within(card as HTMLElement).queryByText("Rogue's Dodge")).not.toBeInTheDocument();
    expect(within(card as HTMLElement).queryByText("Mary")).not.toBeInTheDocument();
    // Invariant: the basic card must stay compact and not introduce the full Daggerheart detail summary.
    expect(within(card as HTMLElement).queryByLabelText("Daggerheart full info")).not.toBeInTheDocument();
  });

  it("renders full mode with web-derived Daggerheart summary content", () => {
    render(<CharacterCard character={characterCardFixtures.full} variant="full" />);

    const statistics = screen.getByLabelText("Character statistics");
    const hopeSummary = screen.getByLabelText("Character hope summary");
    const fullInfo = screen.getByLabelText("Daggerheart full info");

    expect(screen.getByRole("heading", { level: 2, name: "Mira" })).toBeInTheDocument();
    expect(screen.getByText("(she/her)")).toBeInTheDocument();
    expect(screen.getByText("Rogue / Nightwalker")).toBeInTheDocument();
    expect(screen.getByText("Human, Slyborne")).toBeInTheDocument();
    expect(screen.getByText("Mary")).toBeInTheDocument();
    expect(screen.getByText("Starling")).toBeInTheDocument();
    expect(within(statistics).getByText("3/5")).toBeInTheDocument();
    expect(within(statistics).getByText("2/6")).toBeInTheDocument();
    expect(within(statistics).getByText("4/5")).toBeInTheDocument();
    for (const icon of ["hp", "stress", "evasion", "armor"] as const) {
      expect((statistics as HTMLElement).querySelector(`[data-icon="${icon}"]`)).not.toBeNull();
    }
    expect(within(hopeSummary).getByText("2/6")).toBeInTheDocument();
    expect(within(hopeSummary).queryByText("Rogue's Dodge")).not.toBeInTheDocument();
    expect((hopeSummary as HTMLElement).querySelector('[data-icon="hope"]')).not.toBeNull();
    expect(fullInfo).toBeInTheDocument();
    expect(screen.getByText("AGI 2")).toBeInTheDocument();
    expect(within(fullInfo).getByText("Hope Feature")).toBeInTheDocument();
    expect(within(fullInfo).getByText("Rogue's Dodge")).toBeInTheDocument();
    expect(screen.getByText("Arcane Bolt")).toBeInTheDocument();
    expect(screen.queryByText("Details")).not.toBeInTheDocument();
    expect(screen.queryByText("Background")).not.toBeInTheDocument();
    expect(screen.queryByText("Connections")).not.toBeInTheDocument();
  });

  it("falls back to a placeholder portrait when the image source is missing", () => {
    render(<CharacterCard character={characterCardFixtures.partial} variant="full" />);

    expect(
      screen.getByRole("img", {
        name: /portrait placeholder for zara/i,
      }),
    ).toBeInTheDocument();
    expect(screen.getByText("Zara")).toBeInTheDocument();
    expect(screen.queryByText("Experiences")).not.toBeInTheDocument();
  });
});
