import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { BackstageContextCard } from "./BackstageContextCard";

describe("BackstageContextCard", () => {
  it("renders scene name, paused prompt, and reason together", () => {
    render(
      <BackstageContextCard
        sceneName="Sealed Vault"
        pausedPromptText="The ward crackles when either of you nears the seam. What do you do?"
        reason="Clarify how the ward reacts to tools."
      />,
    );

    expect(screen.getByLabelText("Backstage context")).toBeInTheDocument();
    expect(screen.getByText("Sealed Vault")).toBeInTheDocument();
    expect(screen.getByText(/The ward crackles/)).toBeInTheDocument();
    expect(screen.getByText("Clarify how the ward reacts to tools.")).toBeInTheDocument();
  });

  it("renders nothing when there is no context to show", () => {
    const { container } = render(<BackstageContextCard />);
    expect(container).toBeEmptyDOMElement();
  });
});
