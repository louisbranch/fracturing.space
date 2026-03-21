import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { BackstageCompose } from "./BackstageCompose";

describe("BackstageCompose", () => {
  it("uses Backstage-specific copy when enabled", () => {
    render(
      <BackstageCompose
        draft=""
        viewerReady={false}
        onDraftChange={() => {}}
        onSend={() => {}}
        onReadyToggle={() => {}}
      />,
    );

    expect(screen.getByLabelText("Backstage actions")).toHaveClass("bg-base-300");
    expect(screen.getByLabelText("Backstage message input")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Post" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Mark Ready" })).toBeEnabled();
  });

  it("disables the input when OOC is not open", () => {
    render(
      <BackstageCompose
        draft=""
        viewerReady={false}
        disabled
        onDraftChange={() => {}}
        onSend={() => {}}
        onReadyToggle={() => {}}
      />,
    );

    expect(screen.getByLabelText("Backstage message input")).toBeDisabled();
    expect(screen.getByRole("button", { name: "Mark Ready" })).toBeDisabled();
  });

  it("toggles the ready action label and forwards clicks", async () => {
    const user = userEvent.setup();
    const onReadyToggle = vi.fn();

    render(
      <BackstageCompose
        draft="Need one more clarification."
        viewerReady
        onDraftChange={() => {}}
        onSend={() => {}}
        onReadyToggle={onReadyToggle}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Clear Ready" }));
    expect(onReadyToggle).toHaveBeenCalledOnce();
  });
});
