import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { HUDHeader } from "./HUDHeader";
import { hudHeaderFixtures } from "./fixtures";

describe("HUDHeader", () => {
  it("renders the campaign title, back link, and connection label", () => {
    render(
      <HUDHeader
        backURL={hudHeaderFixtures.connected.backURL}
        campaignName={hudHeaderFixtures.connected.campaignName}
        connection={hudHeaderFixtures.connected.connection}
      />,
    );

    expect(screen.getByLabelText("Player HUD header")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Back To Campaign" })).toHaveAttribute(
      "href",
      "/app/campaigns/camp-cliffside",
    );
    expect(screen.getByText("Cliffside Rescue")).toBeInTheDocument();
    expect(screen.getByText("Connected")).toBeInTheDocument();
  });
});
