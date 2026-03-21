import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { onStageFixtureCatalog } from "./fixtures";
import { OnStageSceneCard } from "./OnStageSceneCard";

describe("OnStageSceneCard", () => {
  it("renders scene, GM output, frame, and acting-character context", () => {
    render(
      <OnStageSceneCard
        sceneName={onStageFixtureCatalog.viewerPosted.sceneName}
        sceneDescription={onStageFixtureCatalog.viewerPosted.sceneDescription}
        gmOutputText={onStageFixtureCatalog.viewerPosted.gmOutputText}
        frameText={onStageFixtureCatalog.viewerPosted.frameText}
        actingCharacterNames={onStageFixtureCatalog.viewerPosted.actingCharacterNames}
        status={{
          label: "Your Beat",
          className: "badge-primary badge-soft",
          tooltip: "Commit the next action for your character and yield when you are ready.",
        }}
      />,
    );

    expect(screen.getByLabelText("On-stage scene context")).toBeInTheDocument();
    expect(screen.getByLabelText("On-stage status: Your Beat")).toHaveClass(
      "tooltip",
      "tooltip-left",
    );
    expect(screen.getByText("Sealed Vault")).toBeInTheDocument();
    expect(screen.getByText("Latest GM Output")).toBeInTheDocument();
    expect(screen.getByText("Current Frame")).toBeInTheDocument();
    expect(screen.getByText("Acting Now")).toBeInTheDocument();
    expect(screen.getByText("Aria")).toBeInTheDocument();
  });
});
