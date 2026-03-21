import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { HUDNavbar } from "./HUDNavbar";

describe("HUDNavbar", () => {
  it("renders three navigation items", () => {
    render(<HUDNavbar activeTab="on-stage" onTabChange={() => {}} />);

    expect(screen.getByLabelText("Player HUD navigation")).toBeInTheDocument();
    expect(screen.getByText("On Stage")).toBeInTheDocument();
    expect(screen.getByText("Backstage")).toBeInTheDocument();
    expect(screen.getByText("Side Chat")).toBeInTheDocument();
  });

  it("marks the active tab with aria-current", () => {
    render(<HUDNavbar activeTab="backstage" onTabChange={() => {}} />);

    expect(screen.getByText("Backstage").closest("button")).toHaveAttribute("aria-current", "page");
    expect(screen.getByText("On Stage").closest("button")).not.toHaveAttribute("aria-current");
    expect(screen.getByText("Side Chat").closest("button")).not.toHaveAttribute("aria-current");
  });

  it("shows an update count on inactive tabs with updates", () => {
    render(<HUDNavbar activeTab="on-stage" onTabChange={() => {}} tabsWithUpdates={new Map([["side-chat", 2]])} />);

    const badge = screen.getByText("2");
    expect(badge).toHaveClass("badge-primary");

    // Invariant: active tab should not show an indicator even if listed in tabsWithUpdates
    const onStageButton = screen.getByText("On Stage").closest("button")!;
    expect(onStageButton.closest(".indicator")).toBeNull();
  });

  it("calls onTabChange when a tab is clicked", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<HUDNavbar activeTab="on-stage" onTabChange={onChange} />);

    await user.click(screen.getByText("Side Chat"));
    expect(onChange).toHaveBeenCalledWith("side-chat");
  });

  it("keeps Backstage and Side Chat copy distinct", () => {
    const { container } = render(<HUDNavbar activeTab="on-stage" onTabChange={() => {}} />);

    expect(container.textContent).toContain("Backstage");
    expect(container.textContent).toContain("Side Chat");
    expect(container.textContent).not.toContain("Out-of-character chat");
  });
});
