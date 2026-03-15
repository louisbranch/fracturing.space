import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ActingSetRail } from "./ActingSetRail";
import { actingSetRailFixtures } from "./fixtures";

describe("ActingSetRail", () => {
  it("renders spotlighted and non-spotlighted acting characters", () => {
    render(<ActingSetRail actingSet={actingSetRailFixtures.multiActor} />);

    expect(screen.getByLabelText("Acting set")).toBeInTheDocument();
    expect(screen.getByText("Aria")).toBeInTheDocument();
    expect(screen.getByText("Corin")).toBeInTheDocument();
    expect(screen.getByText("Spotlight")).toBeInTheDocument();
  });
});
