import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ChatSidecar } from "./ChatSidecar";
import { chatSidecarFixtures } from "./fixtures";

describe("ChatSidecar", () => {
  it("renders transcript messages as a secondary surface", () => {
    render(<ChatSidecar messages={chatSidecarFixtures.messages} />);

    expect(screen.getByLabelText("Session chat")).toBeInTheDocument();
    expect(screen.getByText("Human Transcript")).toBeInTheDocument();
    expect(screen.getByText(/Keep the scout talking while the rope holds/i)).toBeInTheDocument();
  });
});
