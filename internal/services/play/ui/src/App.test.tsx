import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { App } from "./App";

describe("App", () => {
  it("renders the Storybook handoff on the root placeholder", () => {
    render(
      <App
        mode={{
          kind: "root-placeholder",
        }}
        shellConfig={{
          campaignId: "",
          bootstrapPath: "",
          realtimePath: "/realtime",
          backURL: "",
        }}
      />,
    );

    expect(screen.getByText("Play runtime UI deferred")).toBeInTheDocument();
    expect(screen.getByText("npm run storybook")).toBeInTheDocument();
    expect(screen.getByText("http://localhost:6006")).toBeInTheDocument();
    expect(screen.getByText("/realtime")).toBeInTheDocument();
  });

  it("renders the runtime placeholder for campaign routes", () => {
    render(
      <App
        mode={{
          kind: "runtime-placeholder",
          campaignId: "guildhouse",
        }}
        shellConfig={{
          campaignId: "guildhouse",
          bootstrapPath: "/api/campaigns/guildhouse/bootstrap",
          realtimePath: "/realtime",
          backURL: "/app/campaigns/guildhouse/game",
        }}
      />,
    );

    expect(screen.getByText("Play runtime UI deferred")).toBeInTheDocument();
    expect(screen.getByText("/campaigns/guildhouse")).toBeInTheDocument();
    expect(screen.getByText("http://localhost:6006")).toBeInTheDocument();
    expect(screen.getByText("/api/campaigns/guildhouse/bootstrap")).toBeInTheDocument();
    expect(screen.getByText("/app/campaigns/guildhouse/game")).toBeInTheDocument();
  });

  it("renders config-missing fallback when runtime mode lacks shell config", () => {
    render(
      <App
        mode={{
          kind: "runtime",
          campaignId: "guildhouse",
        }}
        shellConfig={null}
      />,
    );

    expect(screen.getByText("Play shell config not available")).toBeInTheDocument();
  });

  it("renders the unsupported route screen for unknown paths", () => {
    render(
      <App
        mode={{
          kind: "unsupported",
          path: "/mystery",
        }}
      />,
    );

    expect(screen.getByText("No UI mapped to this path")).toBeInTheDocument();
    expect(screen.getByText("/mystery")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open Storybook" })).toHaveAttribute("href", "http://localhost:6006");
  });
});
