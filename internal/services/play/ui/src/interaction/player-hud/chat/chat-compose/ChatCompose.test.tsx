import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ChatCompose } from "./ChatCompose";

describe("ChatCompose", () => {
  it("disables the send button when draft is empty", () => {
    render(<ChatCompose draft="" onDraftChange={() => {}} onSend={() => {}} />);
    expect(screen.getByRole("button", { name: "Send" })).toBeDisabled();
  });

  it("enables the send button when draft has content", () => {
    render(<ChatCompose draft="hello" onDraftChange={() => {}} onSend={() => {}} />);
    expect(screen.getByRole("button", { name: "Send" })).toBeEnabled();
  });

  it("calls onSend when the send button is clicked", async () => {
    const user = userEvent.setup();
    const onSend = vi.fn();
    render(<ChatCompose draft="hello" onDraftChange={() => {}} onSend={onSend} />);

    await user.click(screen.getByRole("button", { name: "Send" }));
    expect(onSend).toHaveBeenCalledOnce();
  });

  it("calls onSend on Enter key (without Shift)", async () => {
    const user = userEvent.setup();
    const onSend = vi.fn();
    render(<ChatCompose draft="hello" onDraftChange={() => {}} onSend={onSend} />);

    const textarea = screen.getByLabelText("Chat message input");
    await user.click(textarea);
    await user.keyboard("{Enter}");
    expect(onSend).toHaveBeenCalledOnce();
  });

  it("does not call onSend on Shift+Enter", async () => {
    const user = userEvent.setup();
    const onSend = vi.fn();
    render(<ChatCompose draft="hello" onDraftChange={() => {}} onSend={onSend} />);

    const textarea = screen.getByLabelText("Chat message input");
    await user.click(textarea);
    await user.keyboard("{Shift>}{Enter}{/Shift}");
    expect(onSend).not.toHaveBeenCalled();
  });

  it("calls onDraftChange when typing", async () => {
    const user = userEvent.setup();
    const onDraftChange = vi.fn();
    render(<ChatCompose draft="" onDraftChange={onDraftChange} onSend={() => {}} />);

    const textarea = screen.getByLabelText("Chat message input");
    await user.type(textarea, "hi");
    expect(onDraftChange).toHaveBeenCalled();
  });
});
