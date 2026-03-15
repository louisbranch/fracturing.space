import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { OOCOverlayPanel } from "./OOCOverlayPanel";
import { oocOverlayPanelFixtures } from "./fixtures";

describe("OOCOverlayPanel", () => {
  it("renders pause reason, posts, and ready-to-resume badges", () => {
    render(<OOCOverlayPanel phase={oocOverlayPanelFixtures.phase} ooc={oocOverlayPanelFixtures.ooc} />);

    expect(screen.getByLabelText("OOC overlay")).toBeInTheDocument();
    expect(screen.getByText(/Clarify how the ward reacts to touch/i)).toBeInTheDocument();
    expect(screen.getAllByText("Rhea")).toHaveLength(2);
    expect(screen.getByText("Resume Scene")).toBeInTheDocument();
  });
});
