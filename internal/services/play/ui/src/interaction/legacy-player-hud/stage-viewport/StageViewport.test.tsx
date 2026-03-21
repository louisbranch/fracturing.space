import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { StageViewport } from "./StageViewport";
import { stageViewportFixtures } from "./fixtures";

describe("StageViewport", () => {
  it("renders stage content inside an internal scroll container", () => {
    render(<StageViewport stage={stageViewportFixtures.scrolling} />);

    expect(screen.getByLabelText("Player stage viewport")).toBeInTheDocument();
    expect(screen.getByText("Scene: Storm Ledge")).toBeInTheDocument();
    expect(screen.getByText(/The cliff path shudders beneath your boots/i)).toBeInTheDocument();
    expect(document.querySelector(".hud-stage-scroll")).toHaveClass("overflow-y-auto");
  });

  it("renders an empty placeholder when no stage content is present", () => {
    render(<StageViewport stage={stageViewportFixtures.empty} />);

    expect(screen.getByText("[empty for now]")).toBeInTheDocument();
  });
});
