import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { PhaseStatusBanner } from "./PhaseStatusBanner";
import { phaseStatusBannerFixtures } from "./fixtures";

describe("PhaseStatusBanner", () => {
  it("renders player authority with GM authority context", () => {
    render(<PhaseStatusBanner phase={phaseStatusBannerFixtures.players} viewerName="Guide" viewerRole="gm" />);

    expect(screen.getByLabelText("Interaction phase status")).toBeInTheDocument();
    expect(screen.getByText("Players")).toBeInTheDocument();
    expect(screen.getByText(/GM authority:/)).toBeInTheDocument();
    expect(screen.getByText(/Viewing as/)).toBeInTheDocument();
  });

  it("shows the OOC badge when the overlay is open", () => {
    render(<PhaseStatusBanner phase={phaseStatusBannerFixtures.ooc} viewerName="Guide" viewerRole="gm" />);

    expect(screen.getByText("OOC Open")).toBeInTheDocument();
  });
});
