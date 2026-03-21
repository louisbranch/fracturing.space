import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { SideChatPanel } from "./SideChatPanel";
import { emptySideChatState, sideChatState } from "./fixtures";

describe("SideChatPanel", () => {
  it("composes chat list and compose input", () => {
    const { container } = render(
      <SideChatPanel
        state={sideChatState}
        draft=""
        onDraftChange={() => {}}
        onSend={() => {}}
      />,
    );

    const scrollRegion = container.querySelector(".hud-panel-scroll-region");
    expect(scrollRegion).not.toBeNull();
    expect(screen.getByLabelText("Side chat")).toBeInTheDocument();
    expect(screen.getByLabelText("Side chat messages")).toBeInTheDocument();
    expect(screen.getByLabelText("Chat message input")).toBeInTheDocument();
    expect(scrollRegion).toContainElement(screen.getByLabelText("Side chat messages"));
  });

  it("shows empty state when there are no messages", () => {
    render(
      <SideChatPanel
        state={emptySideChatState}
        draft=""
        onDraftChange={() => {}}
        onSend={() => {}}
      />,
    );

    expect(screen.getByText("No messages yet")).toBeInTheDocument();
  });
});
