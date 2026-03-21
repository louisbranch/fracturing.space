import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { backstageFixtureCatalog, playerHUDFixtureCatalog, sideChatState } from "../shared/fixtures";
import { PlayerHUDShell } from "./PlayerHUDShell";

const baseProps = {
  onStage: playerHUDFixtureCatalog.onStage.onStage,
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
    render(
      <PlayerHUDShell
        activeTab="on-stage"
        connectionState="connected"
        campaignNavigation={playerHUDFixtureCatalog.onStage.campaignNavigation}
        isSidebarOpen={false}
        onSidebarOpenChange={() => {}}
        onTabChange={() => {}}
        {...baseProps}
      />,
    );

    expect(screen.getByLabelText("Player HUD shell")).toHaveClass("bg-base-300");
    expect(screen.getByLabelText("Player HUD navigation")).toBeInTheDocument();
    expect(screen.getByLabelText("Connection status: Connected")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "On Stage" })).toBeInTheDocument();
    expect(screen.getByLabelText("On-stage scene context")).toBeInTheDocument();
    expect(screen.getByText("Drowned Chapel Vault")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Expand scene description" })).toBeInTheDocument();
    expect(screen.getByLabelText("On-stage GM interaction")).toBeInTheDocument();
    expect(screen.getByText(/if this goes wrong, i need to know which compromise/i)).toBeInTheDocument();
    expect(screen.getByText(/one moment where the ward's recoil is weakest/i)).toBeInTheDocument();
    expect(screen.getByLabelText("On-stage slot by Ives")).toBeInTheDocument();
    expect(screen.getByText(/he watches the gallery rail for movement/i)).toBeInTheDocument();
    expect(screen.getByLabelText("On-stage action input")).toBeInTheDocument();
    expect(screen.getByLabelText("On-stage participants")).toBeInTheDocument();
    expect(screen.getByLabelText("Guide GM authority")).toBeInTheDocument();
  });

  it("forwards tab changes from the navbar", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <PlayerHUDShell
        activeTab="on-stage"
        connectionState="connected"
        campaignNavigation={playerHUDFixtureCatalog.onStage.campaignNavigation}
        isSidebarOpen={false}
        onSidebarOpenChange={() => {}}
        onTabChange={onChange}
        {...baseProps}
      />,
    );

    await user.click(screen.getByText("Backstage"));
    expect(onChange).toHaveBeenCalledWith("backstage");
  });

  it("forwards sidebar toggle requests from the navbar", async () => {
    const user = userEvent.setup();
    const onSidebarOpenChange = vi.fn();
    render(
      <PlayerHUDShell
        activeTab="on-stage"
        connectionState="connected"
        campaignNavigation={playerHUDFixtureCatalog.onStage.campaignNavigation}
        isSidebarOpen={false}
        onSidebarOpenChange={onSidebarOpenChange}
        onTabChange={() => {}}
        {...baseProps}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Open campaign sidebar" }));
    expect(onSidebarOpenChange).toHaveBeenCalledWith(true);
  });

  it("forwards sidebar close requests from the navbar when the drawer is open", async () => {
    const user = userEvent.setup();
    const onSidebarOpenChange = vi.fn();
    render(
      <PlayerHUDShell
        activeTab="on-stage"
        connectionState="connected"
        campaignNavigation={playerHUDFixtureCatalog.onStage.campaignNavigation}
        isSidebarOpen={true}
        onSidebarOpenChange={onSidebarOpenChange}
        onTabChange={() => {}}
        {...baseProps}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Close campaign sidebar" }));
    expect(onSidebarOpenChange).toHaveBeenCalledWith(false);
  });

  it("renders the side chat panel when activeTab is side-chat", () => {
    render(
      <PlayerHUDShell
        activeTab="side-chat"
        connectionState="connected"
        campaignNavigation={playerHUDFixtureCatalog.sideChat.campaignNavigation}
        isSidebarOpen={false}
        onSidebarOpenChange={() => {}}
        onTabChange={() => {}}
        {...baseProps}
      />,
    );

    expect(screen.getByLabelText("Side chat")).toBeInTheDocument();
    expect(screen.getByLabelText("Side chat messages")).toBeInTheDocument();
    expect(screen.getByLabelText("Chat message input")).toBeInTheDocument();
    expect(screen.getByLabelText("Side chat participants")).toBeInTheDocument();
  });

  it("renders the Backstage panel when activeTab is backstage", () => {
    render(
      <PlayerHUDShell
        activeTab="backstage"
        connectionState="connected"
        campaignNavigation={playerHUDFixtureCatalog.backstage.campaignNavigation}
        isSidebarOpen={false}
        onSidebarOpenChange={() => {}}
        onTabChange={() => {}}
        {...baseProps}
      />,
    );

    expect(screen.getByRole("button", { name: "Backstage" })).toBeInTheDocument();
    expect(screen.getByLabelText("Backstage OOC messages")).toBeInTheDocument();
    expect(screen.getByLabelText("Backstage participants")).toBeInTheDocument();
  });

  it("renders the drawer sidebar when it is open", () => {
    render(
      <PlayerHUDShell
        activeTab="on-stage"
        connectionState="disconnected"
        campaignNavigation={playerHUDFixtureCatalog.onStage.campaignNavigation}
        isSidebarOpen={true}
        onSidebarOpenChange={() => {}}
        onTabChange={() => {}}
        {...baseProps}
      />,
    );

    expect(screen.getByLabelText("Player HUD sidebar")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Return to Campaign" })).toBeInTheDocument();
    expect(screen.getByLabelText("Connection status: Disconnected")).toBeInTheDocument();
  });
});
