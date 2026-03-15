import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { GMReviewPanel } from "./GMReviewPanel";
import { gmReviewPanelFixtures } from "./fixtures";

describe("GMReviewPanel", () => {
  it("summarizes the review counts and actions", () => {
    render(<GMReviewPanel slots={gmReviewPanelFixtures.review} />);

    expect(screen.getByLabelText("GM review panel")).toBeInTheDocument();
    expect(screen.getByText("Under Review")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Accept Phase" })).toBeInTheDocument();
  });
});
