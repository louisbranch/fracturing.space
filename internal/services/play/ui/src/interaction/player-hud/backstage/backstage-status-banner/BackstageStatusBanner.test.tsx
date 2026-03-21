import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { BackstageStatusBanner } from "./BackstageStatusBanner";

describe("BackstageStatusBanner", () => {
  it("shows dormant copy and disables the ready button when OOC is closed", () => {
    render(
      <BackstageStatusBanner
        mode="dormant"
        resumeState="inactive"
        viewerReady={false}
        onViewerReadyToggle={() => {}}
      />,
    );

    expect(screen.getByText("Backstage Idle")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Mark Ready" })).toBeDisabled();
  });

  it("shows terse collecting-ready status copy when OOC is open", () => {
    render(
      <BackstageStatusBanner
        mode="open"
        resumeState="collecting-ready"
        viewerReady={false}
        onViewerReadyToggle={() => {}}
      />,
    );

    expect(screen.getByText("Awaiting player readiness.")).toBeInTheDocument();
  });

  it("shows terse waiting-on-gm status copy when all players are ready", () => {
    render(
      <BackstageStatusBanner
        mode="open"
        resumeState="waiting-on-gm"
        viewerReady={true}
        onViewerReadyToggle={() => {}}
      />,
    );

    expect(screen.getByText("Waiting on GM.")).toBeInTheDocument();
  });

  it("forwards ready-toggle clicks when the ready button is enabled", async () => {
    const user = userEvent.setup();
    const onToggle = vi.fn();

    render(
      <BackstageStatusBanner
        mode="open"
        resumeState="collecting-ready"
        viewerReady={false}
        onViewerReadyToggle={onToggle}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Mark Ready" }));
    expect(onToggle).toHaveBeenCalledOnce();
  });
});
