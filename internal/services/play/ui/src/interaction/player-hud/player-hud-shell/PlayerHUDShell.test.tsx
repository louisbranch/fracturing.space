import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { backstageFixtureCatalog, onStageFixtureCatalog, sideChatState } from "../shared/fixtures";
import { PlayerHUDShell } from "./PlayerHUDShell";

const baseProps = {
  onStage: onStageFixtureCatalog.viewerPosted,
  onStageDraft: "Aria commits to the vault seam.",
  onOnStageDraftChange: () => {},
  onOnStageSubmit: () => {},
  onOnStageSubmitAndYield: () => {},
  onOnStageYield: () => {},
  onOnStageUnyield: () => {},
  backstage: backstageFixtureCatalog.openDiscussion,
  backstageDraft: "",
  onBackstageDraftChange: () => {},
  onBackstageSend: () => {},
  onBackstageReadyToggle: () => {},
  sideChat: sideChatState,
  sideChatDraft: "",
  onSideChatDraftChange: () => {},
  onSideChatSend: () => {},
};

describe("PlayerHUDShell", () => {
  it("assembles the navbar and on-stage panel into a viewport", () => {
    render(<PlayerHUDShell activeTab="on-stage" onTabChange={() => {}} {...baseProps} />);

    expect(screen.getByLabelText("Player HUD shell")).toBeInTheDocument();
    expect(screen.getByLabelText("Player HUD navigation")).toBeInTheDocument();
    expect(screen.getByLabelText("On Stage")).toBeInTheDocument();
    expect(screen.getByLabelText("On-stage scene context")).toBeInTheDocument();
    expect(screen.getByLabelText("On-stage action input")).toBeInTheDocument();
    expect(screen.getByLabelText("On-stage participants")).toBeInTheDocument();
    expect(screen.getByLabelText("Guide GM authority")).toBeInTheDocument();
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
    expect(screen.getByLabelText("Side chat participants")).toBeInTheDocument();
  });

  it("renders the Backstage panel when activeTab is backstage", () => {
    render(<PlayerHUDShell activeTab="backstage" onTabChange={() => {}} {...baseProps} />);

    expect(screen.getByLabelText("Backstage")).toBeInTheDocument();
    expect(screen.getByLabelText("Backstage OOC messages")).toBeInTheDocument();
    expect(screen.getByLabelText("Backstage participants")).toBeInTheDocument();
  });
});
