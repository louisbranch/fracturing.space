import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { interactionComponentFixtures } from "../shared/fixtures";
import { SceneFramePanel } from "./SceneFramePanel";
import { sceneFramePanelFixtures } from "./fixtures";

describe("SceneFramePanel", () => {
  it("renders active scene content with committed GM output and frame text", () => {
    render(<SceneFramePanel phase={interactionComponentFixtures.phase.players} scene={sceneFramePanelFixtures.activeScene} />);

    expect(screen.getByLabelText("Active scene frame")).toBeInTheDocument();
    expect(screen.getByRole("heading", { level: 2, name: "Storm Ledge" })).toBeInTheDocument();
    expect(screen.getByText(/Committed GM Output/)).toBeInTheDocument();
    expect(screen.getByText(/Current Player Frame/)).toBeInTheDocument();
  });

  it("renders a fallback when no scene is selected", () => {
    render(<SceneFramePanel phase={interactionComponentFixtures.phase.players} />);

    expect(screen.getByRole("heading", { level: 2, name: "No active scene" })).toBeInTheDocument();
  });
});
