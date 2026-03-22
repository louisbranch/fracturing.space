import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { onStageStatusBadge } from "../../shared/view-models";
import { archerGuardIllustration, onStageFixtureCatalog } from "./fixtures";
import { OnStageGMInteractionCard } from "./OnStageGMInteractionCard";

describe("OnStageGMInteractionCard", () => {
  it("renders the current interaction illustration with the derived on-stage status", () => {
    render(
      <OnStageGMInteractionCard
        currentInteraction={onStageFixtureCatalog.viewerPosted.currentInteraction}
        interactionHistory={onStageFixtureCatalog.viewerPosted.interactionHistory}
        currentStatus={onStageStatusBadge(onStageFixtureCatalog.viewerPosted)}
      />,
    );

    expect(screen.getByLabelText("On-stage GM interaction")).toBeInTheDocument();
    expect(screen.getByText("At the Vault Seam")).toBeInTheDocument();
    expect(screen.getByLabelText("On-stage interaction status: Your Beat")).toBeInTheDocument();
    expect(screen.getByAltText(/storm lantern burning in darkness/i)).toBeInTheDocument();
  });

  it("renders a compact illustration with its caption when provided", () => {
    const currentInteraction = onStageFixtureCatalog.viewerPosted.currentInteraction;
    if (!currentInteraction) {
      throw new Error("expected current interaction fixture");
    }

    render(
      <OnStageGMInteractionCard
        currentInteraction={{
          ...currentInteraction,
          illustration: archerGuardIllustration,
        }}
        interactionHistory={onStageFixtureCatalog.viewerPosted.interactionHistory}
        currentStatus={onStageStatusBadge(onStageFixtureCatalog.viewerPosted)}
      />,
    );

    expect(screen.getByAltText(/archer guard drawing and aiming/i)).toBeInTheDocument();
    expect(screen.getByText("Enemy attack illustration example.")).toBeInTheDocument();
  });

  it("renders without a media shell when the current interaction has no illustration", () => {
    const currentInteraction = onStageFixtureCatalog.viewerPosted.currentInteraction;
    if (!currentInteraction) {
      throw new Error("expected current interaction fixture");
    }

    render(
      <OnStageGMInteractionCard
        currentInteraction={{
          ...currentInteraction,
          illustration: undefined,
        }}
        interactionHistory={onStageFixtureCatalog.viewerPosted.interactionHistory}
        currentStatus={onStageStatusBadge(onStageFixtureCatalog.viewerPosted)}
      />,
    );

    // Invariant: illustration-free interactions must not expose empty media UI.
    expect(screen.queryByRole("img")).not.toBeInTheDocument();
  });

  it("navigates into older interactions and marks them concluded", async () => {
    const user = userEvent.setup();

    render(
      <OnStageGMInteractionCard
        currentInteraction={onStageFixtureCatalog.viewerPosted.currentInteraction}
        interactionHistory={onStageFixtureCatalog.viewerPosted.interactionHistory}
        currentStatus={onStageStatusBadge(onStageFixtureCatalog.viewerPosted)}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Show older interaction" }));

    expect(screen.getByText("The Warning Lattice")).toBeInTheDocument();
    expect(screen.getByLabelText("On-stage interaction status: Concluded")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Show newer interaction" })).toBeInTheDocument();
  });
});
