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
          indicator: "none",
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

  it("shows a loading bar for pending on-stage statuses", () => {
    render(
      <OnStageSceneCard
        sceneName={onStageFixtureCatalog.waitingOnGM.sceneName}
        sceneDescription={onStageFixtureCatalog.waitingOnGM.sceneDescription}
        gmOutputText={onStageFixtureCatalog.waitingOnGM.gmOutputText}
        actingCharacterNames={onStageFixtureCatalog.waitingOnGM.actingCharacterNames}
        status={{
          label: "Waiting",
          className: "badge-ghost",
          indicator: "loading-bars",
          tooltip: "Waiting for the GM to frame the next beat.",
        }}
      />,
    );

    const status = screen.getByLabelText("On-stage status: Waiting");
    expect(status.querySelector(".loading.loading-bars")).not.toBeNull();
    expect(screen.getByText("Waiting")).toBeInTheDocument();
  });
});
