import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ChatList } from "./ChatList";
import { sideChatMessages, sideChatParticipants } from "./fixtures";

describe("ChatList", () => {
  it("renders a scrollable container with all messages", () => {
    render(
      <ChatList
        messages={sideChatMessages}
        participants={sideChatParticipants}
        viewerParticipantId="p-viewer"
      />,
    );

    const container = screen.getByLabelText("Side chat messages");
    expect(container).toBeInTheDocument();
    expect(container.className).toContain("overflow-y-auto");

    // All message bodies are rendered.
    expect(screen.getByText("Ready when you are.")).toBeInTheDocument();
    expect(screen.getByText("Copy. Moving to the bridge.")).toBeInTheDocument();
  });

  it("groups consecutive messages: name on first, avatar on last in run", () => {
    render(
      <ChatList
        messages={sideChatMessages}
        participants={sideChatParticipants}
        viewerParticipantId="p-viewer"
      />,
    );

    // Corin has two consecutive messages (m1, m2). Name should appear once.
    const corinHeaders = screen.getAllByText("Corin");
    expect(corinHeaders).toHaveLength(1);
  });

  it("aligns viewer messages to chat-end and others to chat-start", () => {
    const { container } = render(
      <ChatList
        messages={sideChatMessages}
        participants={sideChatParticipants}
        viewerParticipantId="p-viewer"
      />,
    );

    const endBubbles = container.querySelectorAll(".chat-end");
    const startBubbles = container.querySelectorAll(".chat-start");

    // Viewer has 3 messages (m3, m5, m6); others have 3 (m1, m2, m4).
    expect(endBubbles).toHaveLength(3);
    expect(startBubbles).toHaveLength(3);
  });

  it("shows empty state when no messages", () => {
    render(
      <ChatList
        messages={[]}
        participants={sideChatParticipants}
        viewerParticipantId="p-viewer"
      />,
    );

    expect(screen.getByText("No messages yet")).toBeInTheDocument();
  });
});
