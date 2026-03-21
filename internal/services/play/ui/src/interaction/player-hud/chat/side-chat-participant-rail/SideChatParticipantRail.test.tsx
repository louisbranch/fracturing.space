import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { SideChatParticipantRail } from "./SideChatParticipantRail";
import { sideChatState } from "./fixtures";

describe("SideChatParticipantRail", () => {
  it("renders the side chat participant rail", () => {
    render(
      <SideChatParticipantRail
        participants={sideChatState.participants}
        viewerParticipantId={sideChatState.viewerParticipantId}
      />,
    );

    expect(screen.getByLabelText("Side chat participants")).toBeInTheDocument();
    expect(screen.getByLabelText("Bryn: typing")).toBeInTheDocument();
    expect(screen.getByLabelText("Guide: waiting")).toBeInTheDocument();
    expect(screen.getAllByText("PLAYER")).toHaveLength(2);
    expect(screen.getByText("GM")).toBeInTheDocument();
  });
});
