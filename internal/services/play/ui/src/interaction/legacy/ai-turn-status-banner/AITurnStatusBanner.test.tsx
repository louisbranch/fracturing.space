import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AITurnStatusBanner } from "./AITurnStatusBanner";
import { aiTurnStatusBannerFixtures } from "./fixtures";

describe("AITurnStatusBanner", () => {
  it("renders retry affordance for failed turns", () => {
    render(<AITurnStatusBanner aiTurn={aiTurnStatusBannerFixtures.failed} />);

    expect(screen.getByLabelText("AI turn status")).toBeInTheDocument();
    expect(screen.getByText("Failed")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Retry AI Turn" })).toBeInTheDocument();
  });
});
