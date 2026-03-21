import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { BackstageCompose } from "./BackstageCompose";

describe("BackstageCompose", () => {
  it("uses Backstage-specific copy when enabled", () => {
    render(
      <BackstageCompose
        draft=""
        onDraftChange={() => {}}
        onSend={() => {}}
      />,
    );

    expect(screen.getByLabelText("Backstage message input")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Post" })).toBeDisabled();
  });

  it("disables the input when OOC is not open", () => {
    render(
      <BackstageCompose
        draft=""
        disabled
        onDraftChange={() => {}}
        onSend={() => {}}
      />,
    );

    expect(screen.getByLabelText("Backstage message input")).toBeDisabled();
  });
});
