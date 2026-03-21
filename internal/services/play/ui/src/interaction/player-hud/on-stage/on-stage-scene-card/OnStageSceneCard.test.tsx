import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it } from "vitest";
import { vi } from "vitest";
import { onStageFixtureCatalog } from "./fixtures";
import { OnStageSceneCard } from "./OnStageSceneCard";

describe("OnStageSceneCard", () => {
  it("renders the active scene header and scene portraits", () => {
    render(
      <OnStageSceneCard
        sceneName={onStageFixtureCatalog.viewerPosted.scene.name}
        sceneDescription={onStageFixtureCatalog.viewerPosted.scene.description}
        sceneCharacters={onStageFixtureCatalog.viewerPosted.scene.characters}
        resolvedInteractionCount={onStageFixtureCatalog.viewerPosted.scene.resolvedInteractionCount}
        expanded
        onToggle={() => {}}
      />,
    );

    expect(screen.getByLabelText("On-stage scene context")).toHaveClass("bg-base-300");
    expect(screen.getByText("Sealed Vault")).toBeInTheDocument();
    expect(screen.getByText("Active Scene")).toBeInTheDocument();
    expect(screen.getByLabelText("Scene characters: Aria, Corin")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Inspect Aria" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Inspect Corin" })).toBeInTheDocument();
  });

  it("shows collapsed scene descriptions with an expand control and allows reopening them", async () => {
    const user = userEvent.setup();

    function Preview() {
      const [expanded, setExpanded] = useState(false);
      return (
        <OnStageSceneCard
          sceneName={onStageFixtureCatalog.viewerPosted.scene.name}
          sceneDescription={onStageFixtureCatalog.viewerPosted.scene.description}
          sceneCharacters={onStageFixtureCatalog.viewerPosted.scene.characters}
          resolvedInteractionCount={onStageFixtureCatalog.viewerPosted.scene.resolvedInteractionCount}
          expanded={expanded}
          onToggle={() => setExpanded((current) => !current)}
        />
      );
    }

    render(<Preview />);

    expect(screen.getByRole("button", { name: "Expand scene description" })).toBeInTheDocument();
    expect(screen.getByText(/^A humming ward seals the old vault/i)).toHaveClass("truncate");

    await user.click(screen.getByRole("button", { name: "Expand scene description" }));
    expect(screen.getByRole("button", { name: "Collapse scene description" })).toBeInTheDocument();
    expect(screen.getByText(/hairline seam catches the lantern light/i)).toBeInTheDocument();
  });

  it("forwards scene portrait clicks when character inspection is enabled", async () => {
    const user = userEvent.setup();
    const onCharacterInspect = vi.fn();

    render(
      <OnStageSceneCard
        sceneName={onStageFixtureCatalog.viewerPosted.scene.name}
        sceneDescription={onStageFixtureCatalog.viewerPosted.scene.description}
        sceneCharacters={onStageFixtureCatalog.viewerPosted.scene.characters}
        resolvedInteractionCount={onStageFixtureCatalog.viewerPosted.scene.resolvedInteractionCount}
        expanded
        onToggle={() => {}}
        onCharacterInspect={onCharacterInspect}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Inspect Aria" }));
    expect(onCharacterInspect).toHaveBeenCalledWith("char-aria");
  });
});
