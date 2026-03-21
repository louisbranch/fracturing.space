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
        status={{
          label: "OOC Open",
          className: "badge-warning badge-soft",
          indicator: "none",
          tooltip: "Awaiting player readiness.",
        }}
      />,
    );

    expect(screen.getByLabelText("Backstage context")).toHaveClass("bg-base-300");
    expect(screen.getByLabelText("Backstage status: OOC Open")).toHaveClass(
      "tooltip",
      "tooltip-left",
    );
    const sceneName = screen.getByText("Sealed Vault");
    const pausedSceneBadge = screen.getByText("Paused Scene");
    expect(sceneName).toBeInTheDocument();
    expect(sceneName.compareDocumentPosition(pausedSceneBadge) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(screen.getByText(/The ward crackles/)).toBeInTheDocument();
    expect(screen.getByText("Clarify how the ward reacts to tools.")).toBeInTheDocument();
  });

  it("renders nothing when there is no context to show", () => {
    const { container } = render(
      <BackstageContextCard
        status={{
          label: "Backstage Idle",
          className: "badge-ghost",
          indicator: "none",
          tooltip: "OOC is closed.",
        }}
      />,
    );
    expect(container).toBeEmptyDOMElement();
  });

  it("shows a loading bar for waiting-on-gm backstage status", () => {
    render(
      <BackstageContextCard
        sceneName="Sealed Vault"
        pausedPromptText="The ward crackles when either of you nears the seam. What do you do?"
        status={{
          label: "Waiting on GM",
          className: "badge-info badge-soft",
          indicator: "loading-bars",
          tooltip: "All players are ready. Waiting for the GM to resume on-stage play.",
        }}
      />,
    );

    const status = screen.getByLabelText("Backstage status: Waiting on GM");
    expect(status.querySelector(".loading.loading-bars")).not.toBeNull();
    expect(screen.getByText("Waiting on GM")).toBeInTheDocument();
  });
});
