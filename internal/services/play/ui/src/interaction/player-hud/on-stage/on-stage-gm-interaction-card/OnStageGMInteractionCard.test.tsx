import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { onStageStatusBadge } from "../../shared/view-models";
import { onStageFixtureCatalog } from "./fixtures";
import { OnStageGMInteractionCard } from "./OnStageGMInteractionCard";

describe("OnStageGMInteractionCard", () => {
  it("renders the current interaction with the derived on-stage status", () => {
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
