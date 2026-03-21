import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { HUDConnectionBadge } from "./HUDConnectionBadge";

describe("HUDConnectionBadge", () => {
  it("renders the connected state with the success badge styling", () => {
    render(<HUDConnectionBadge connectionState="connected" />);

    const badge = screen.getByLabelText("Connection status: Connected");
    expect(badge).toHaveClass("tooltip", "tooltip-left");
    expect(badge).toHaveAttribute("data-tip", "Realtime connection is live.");
    expect(badge.firstElementChild).toHaveClass("badge-success", "badge-soft");
  });

  it("renders the reconnecting state with the animation class", () => {
    const { container } = render(<HUDConnectionBadge connectionState="reconnecting" />);

    const badge = screen.getByLabelText("Connection status: Reconnecting");
    expect(badge).toHaveAttribute("data-tip", "Attempting to restore realtime updates.");
    expect(badge.firstElementChild).toHaveClass("badge-warning", "badge-soft");
    expect(container.querySelector(".animate-spin")).not.toBeNull();
  });

  it("renders the disconnected state with the error badge styling", () => {
    render(<HUDConnectionBadge connectionState="disconnected" />);

    const badge = screen.getByLabelText("Connection status: Disconnected");
    expect(badge).toHaveAttribute("data-tip", "Realtime connection is unavailable.");
    expect(badge.firstElementChild).toHaveClass("badge-error", "badge-soft");
  });
});
