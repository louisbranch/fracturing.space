import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { onStageFixtureCatalog } from "./fixtures";
import { OnStageCompose } from "./OnStageCompose";

describe("OnStageCompose", () => {
  it("shows the player action controls while acting", () => {
    render(
      <OnStageCompose
        draft="Aria hooks the pry tool into the seam."
        controls={onStageFixtureCatalog.viewerPosted.viewerControls}
        mechanicsExtension={onStageFixtureCatalog.viewerPosted.mechanicsExtension}
        onDraftChange={() => {}}
        onSubmit={() => {}}
        onSubmitAndYield={() => {}}
        onYield={() => {}}
        onUnyield={() => {}}
      />,
    );

    expect(screen.getByLabelText("On-stage action input")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Submit" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Submit & Yield" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Yield" })).toBeInTheDocument();
    expect(screen.queryByText("System actions")).not.toBeInTheDocument();
  });

  it("switches to an unyield action after the viewer has already yielded", async () => {
    const user = userEvent.setup();
    const onUnyield = vi.fn();

    render(
      <OnStageCompose
        draft="Aria hooks the pry tool into the seam."
        controls={onStageFixtureCatalog.yieldedWaiting.viewerControls}
        mechanicsExtension={onStageFixtureCatalog.yieldedWaiting.mechanicsExtension}
        onDraftChange={() => {}}
        onSubmit={() => {}}
        onSubmitAndYield={() => {}}
        onYield={() => {}}
        onUnyield={onUnyield}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Unyield" }));
    expect(onUnyield).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("button", { name: "Submit" })).not.toBeInTheDocument();
  });

  it("disables the input when on-stage actions are blocked", () => {
    render(
      <OnStageCompose
        draft=""
        controls={onStageFixtureCatalog.oocBlocked.viewerControls}
        mechanicsExtension={onStageFixtureCatalog.oocBlocked.mechanicsExtension}
        onDraftChange={() => {}}
        onSubmit={() => {}}
        onSubmitAndYield={() => {}}
        onYield={() => {}}
        onUnyield={() => {}}
      />,
    );

    expect(screen.getByLabelText("On-stage action input")).toBeDisabled();
    expect(screen.getByPlaceholderText("Commit the next action for your character...")).toBeInTheDocument();
    expect(
      screen.queryByText("Backstage OOC is open. Resolve the ruling there before acting on-stage."),
    ).not.toBeInTheDocument();
  });
});
