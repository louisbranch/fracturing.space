import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { sideChatState } from "../shared/fixtures";
import { PlayerHUDShell } from "./PlayerHUDShell";

const baseProps = {
  sideChat: sideChatState,
  sideChatDraft: "",
  onSideChatDraftChange: () => {},
  onSideChatSend: () => {},
};

describe("PlayerHUDShell", () => {
  it("assembles the navbar and content placeholder into a viewport", () => {
    render(<PlayerHUDShell activeTab="on-stage" onTabChange={() => {}} {...baseProps} />);

    expect(screen.getByLabelText("Player HUD shell")).toBeInTheDocument();
    expect(screen.getByLabelText("Player HUD navigation")).toBeInTheDocument();
    expect(screen.getByText("on-stage — content coming soon")).toBeInTheDocument();
  });

  it("forwards tab changes from the navbar", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<PlayerHUDShell activeTab="on-stage" onTabChange={onChange} {...baseProps} />);

    await user.click(screen.getByText("Backstage"));
    expect(onChange).toHaveBeenCalledWith("backstage");
  });

  it("renders the side chat panel when activeTab is side-chat", () => {
    render(<PlayerHUDShell activeTab="side-chat" onTabChange={() => {}} {...baseProps} />);

    expect(screen.getByLabelText("Side chat")).toBeInTheDocument();
    expect(screen.getByLabelText("Side chat messages")).toBeInTheDocument();
    expect(screen.getByLabelText("Chat message input")).toBeInTheDocument();
    // Placeholder should not be shown.
    expect(screen.queryByText(/content coming soon/)).not.toBeInTheDocument();
  });
});
