import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { PlayerSlotBoard } from "./PlayerSlotBoard";
import { playerSlotBoardFixtures } from "./fixtures";

describe("PlayerSlotBoard", () => {
  it("renders slot summaries and review badges", () => {
    render(<PlayerSlotBoard slots={playerSlotBoardFixtures.review} />);

    expect(screen.getByLabelText("Player slot board")).toBeInTheDocument();
    expect(screen.getAllByText("Under Review")).toHaveLength(2);
    expect(screen.getByText(/Aria darts for the loose mooring pin/i)).toBeInTheDocument();
  });

  it("renders revision feedback when present", () => {
    render(<PlayerSlotBoard slots={playerSlotBoardFixtures.revisions} />);

    expect(screen.getByText("Changes Requested")).toBeInTheDocument();
    expect(screen.getByText(/Keep the lantern dry/i)).toBeInTheDocument();
  });
});
