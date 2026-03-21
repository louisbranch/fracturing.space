import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { BackstagePanel } from "./BackstagePanel";
import { backstageFixtureCatalog } from "./fixtures";

describe("BackstagePanel", () => {
  it("assembles the Backstage context, transcript, and compose regions", () => {
    render(
      <BackstagePanel
        state={backstageFixtureCatalog.openDiscussion}
        draft=""
        onDraftChange={() => {}}
        onSend={() => {}}
        onReadyToggle={() => {}}
      />,
    );

    expect(screen.getByLabelText("Backstage")).toBeInTheDocument();
    expect(screen.getByLabelText("Backstage context")).toBeInTheDocument();
    expect(screen.getByLabelText("Backstage status: OOC Open")).toBeInTheDocument();
    expect(screen.getByLabelText("Backstage OOC messages")).toBeInTheDocument();
    expect(screen.getByLabelText("Backstage message input")).toBeInTheDocument();
  });

  it("keeps the context card visible and disables actions when Backstage is dormant", () => {
    render(
      <BackstagePanel
        state={backstageFixtureCatalog.dormant}
        draft=""
        onDraftChange={() => {}}
        onSend={() => {}}
        onReadyToggle={() => {}}
      />,
    );

    expect(screen.getByLabelText("Backstage context")).toBeInTheDocument();
    expect(screen.getByLabelText("Backstage status: Backstage Idle")).toBeInTheDocument();
    expect(screen.getByLabelText("Backstage message input")).toBeDisabled();
    expect(screen.getByRole("button", { name: "Mark Ready" })).toBeDisabled();
  });

  it("forwards ready-toggle clicks from the compose controls", async () => {
    const user = userEvent.setup();
    const onReadyToggle = vi.fn();

    render(
      <BackstagePanel
        state={backstageFixtureCatalog.openDiscussion}
        draft=""
        onDraftChange={() => {}}
        onSend={() => {}}
        onReadyToggle={onReadyToggle}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Mark Ready" }));
    expect(onReadyToggle).toHaveBeenCalledOnce();
  });
});
