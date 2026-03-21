import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ChatBubble } from "./ChatBubble";

describe("ChatBubble", () => {
  it("renders with chat-end alignment for the viewer's own messages", () => {
    const { container } = render(
      <ChatBubble body="Hello" time="12:34" alignment="end" />,
    );
    expect(container.querySelector(".chat-end")).toBeInTheDocument();
  });

  it("renders with chat-start alignment for other participants", () => {
    const { container } = render(
      <ChatBubble body="Hello" time="12:34" alignment="start" />,
    );
    expect(container.querySelector(".chat-start")).toBeInTheDocument();
  });

  it("shows participant name only when showName is provided", () => {
    const { rerender } = render(
      <ChatBubble body="Hello" time="12:34" alignment="start" showName="Corin" />,
    );
    expect(screen.getByText("Corin")).toBeInTheDocument();

    rerender(<ChatBubble body="Hello" time="12:34" alignment="start" />);
    expect(screen.queryByText("Corin")).not.toBeInTheDocument();
  });

  it("shows avatar content only when showAvatar is true, but always reserves space", () => {
    const { container, rerender } = render(
      <ChatBubble
        body="Hello"
        time="12:34"
        alignment="start"
        showAvatar
        avatarFallback="C"
      />,
    );
    expect(container.querySelector(".chat-image")).toBeInTheDocument();
    expect(screen.getByText("C")).toBeInTheDocument();

    rerender(<ChatBubble body="Hello" time="12:34" alignment="start" />);
    // Spacer div still present for alignment, but no avatar content.
    expect(container.querySelector(".chat-image")).toBeInTheDocument();
    expect(screen.queryByText("C")).not.toBeInTheDocument();
  });

  it("renders the timestamp inside the bubble", () => {
    render(<ChatBubble body="Hello" time="16:30" alignment="start" />);
    expect(screen.getByText("16:30")).toBeInTheDocument();
  });
});
