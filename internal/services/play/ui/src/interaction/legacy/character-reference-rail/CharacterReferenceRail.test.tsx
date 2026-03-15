import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { CharacterReferenceRail } from "./CharacterReferenceRail";
import { characterReferenceRailFixtures } from "./fixtures";

describe("CharacterReferenceRail", () => {
  it("renders character cards and the selected character sheet", () => {
    render(<CharacterReferenceRail {...characterReferenceRailFixtures} />);

    expect(screen.getByLabelText("Character reference rail")).toBeInTheDocument();
    expect(screen.getByText("Scene Roster")).toBeInTheDocument();
    expect(screen.getAllByText("Aria").length).toBeGreaterThan(0);
    expect(screen.getByText("Selected Character Sheet")).toBeInTheDocument();
  });
});
