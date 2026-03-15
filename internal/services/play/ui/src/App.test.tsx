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
      />,
    );

    expect(screen.getByText("Play runtime UI deferred")).toBeInTheDocument();
    expect(screen.getByText("npm run storybook")).toBeInTheDocument();
    expect(screen.getByText("http://localhost:6006")).toBeInTheDocument();
  });

  it("renders the runtime placeholder for campaign routes", () => {
    render(
      <App
        mode={{
          kind: "runtime-placeholder",
          campaignId: "guildhouse",
        }}
      />,
    );

    expect(screen.getByText("Play runtime UI deferred")).toBeInTheDocument();
    expect(screen.getByText("/campaigns/guildhouse")).toBeInTheDocument();
    expect(screen.getByText("http://localhost:6006")).toBeInTheDocument();
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
