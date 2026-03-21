import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { onStageFixtureCatalog } from "../shared/fixtures";
import { OnStagePanel } from "./OnStagePanel";

describe("OnStagePanel", () => {
  it("renders the on-stage scene context, embedded status badge, slot list, and compose regions", () => {
    render(
      <OnStagePanel
        state={onStageFixtureCatalog.viewerPosted}
        draft="Aria hooks the pry tool into the seam."
        onDraftChange={() => {}}
        onSubmit={() => {}}
        onSubmitAndYield={() => {}}
        onYield={() => {}}
        onUnyield={() => {}}
      />,
    );

    expect(screen.getByLabelText("On Stage")).toBeInTheDocument();
    expect(screen.getByLabelText("On-stage scene context")).toBeInTheDocument();
    expect(screen.getByLabelText("On-stage status: Your Beat")).toBeInTheDocument();
    expect(screen.getByLabelText("On-stage messages")).toBeInTheDocument();
    expect(screen.getByLabelText("On-stage actions")).toBeInTheDocument();
    expect(screen.getByLabelText("On-stage slot by Rhea")).toBeInTheDocument();
    expect(screen.getByLabelText("On-stage slot by Bryn")).toBeInTheDocument();
    expect(screen.getByLabelText("Characters: Aria")).toBeInTheDocument();
    expect(screen.getByLabelText("Characters: Corin")).toBeInTheDocument();
  });

  it("shows the OOC-blocked state as informational and non-actionable", () => {
    render(
      <OnStagePanel
        state={onStageFixtureCatalog.oocBlocked}
        draft=""
        onDraftChange={() => {}}
        onSubmit={() => {}}
        onSubmitAndYield={() => {}}
        onYield={() => {}}
        onUnyield={() => {}}
      />,
    );

    expect(screen.getByLabelText("On-stage status: OOC Open")).toBeInTheDocument();
    expect(screen.getByLabelText("On-stage action input")).toBeDisabled();
    expect(screen.queryByRole("button", { name: "Submit" })).not.toBeInTheDocument();
  });

  it("forwards submit, submit-and-yield, yield, and unyield actions", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    const onSubmitAndYield = vi.fn();
    const onYield = vi.fn();
    const onUnyield = vi.fn();

    const { rerender } = render(
      <OnStagePanel
        state={onStageFixtureCatalog.viewerPosted}
        draft="Aria hooks the pry tool into the seam."
        onDraftChange={() => {}}
        onSubmit={onSubmit}
        onSubmitAndYield={onSubmitAndYield}
        onYield={onYield}
        onUnyield={onUnyield}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Submit" }));
    await user.click(screen.getByRole("button", { name: "Submit & Yield" }));
    await user.click(screen.getByRole("button", { name: "Yield" }));

    expect(onSubmit).toHaveBeenCalledTimes(1);
    expect(onSubmitAndYield).toHaveBeenCalledTimes(1);
    expect(onYield).toHaveBeenCalledTimes(1);

    rerender(
      <OnStagePanel
        state={onStageFixtureCatalog.yieldedWaiting}
        draft="Aria hooks the pry tool into the seam."
        onDraftChange={() => {}}
        onSubmit={onSubmit}
        onSubmitAndYield={onSubmitAndYield}
        onYield={onYield}
        onUnyield={onUnyield}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Unyield" }));
    expect(onUnyield).toHaveBeenCalledTimes(1);
  });

  it("keeps the previous committed text visible when revisions are requested", () => {
    render(
      <OnStagePanel
        state={onStageFixtureCatalog.changesRequested}
        draft="Aria slides the tool into the seam without touching it."
        onDraftChange={() => {}}
        onSubmit={() => {}}
        onSubmitAndYield={() => {}}
        onYield={() => {}}
        onUnyield={() => {}}
      />,
    );

    expect(
      screen.getByText("Aria hooks a pry tool into the seam and braces for the ward's recoil."),
    ).toBeInTheDocument();
    expect(screen.getByLabelText("Revision request")).toBeInTheDocument();
    expect(screen.getByText("Commit to how Aria keeps contact off the seam itself.")).toBeInTheDocument();
  });
});
