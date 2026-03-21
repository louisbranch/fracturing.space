import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { SideChatPanel } from "./SideChatPanel";
import { emptySideChatState, sideChatState } from "./fixtures";

describe("SideChatPanel", () => {
  it("composes chat list and compose input", () => {
    render(
      <SideChatPanel
        state={sideChatState}
        draft=""
        onDraftChange={() => {}}
        onSend={() => {}}
      />,
    );

    expect(screen.getByLabelText("Side chat")).toBeInTheDocument();
    expect(screen.getByLabelText("Side chat messages")).toBeInTheDocument();
    expect(screen.getByLabelText("Chat message input")).toBeInTheDocument();
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
