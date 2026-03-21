import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ParticipantPortraitRail } from "./ParticipantPortraitRail";
import { participantPortraitRailFixtures } from "./fixtures";

describe("ParticipantPortraitRail", () => {
  it("renders participant portrait labels, tooltip-backed status, and GM authority", () => {
    render(
      <ParticipantPortraitRail
        participants={participantPortraitRailFixtures.ready}
        viewerParticipantId="p-rhea"
        ariaLabel="Backstage participants"
      />,
    );

    expect(screen.getByLabelText("Backstage participants")).toBeInTheDocument();
    expect(screen.getByLabelText("Rhea: ready")).toBeInTheDocument();
    expect(screen.getByLabelText("Rhea status: Ready to resume")).toBeInTheDocument();
    expect(screen.getByLabelText("Guide: waiting")).toBeInTheDocument();
    expect(screen.getByLabelText("Guide GM authority")).toBeInTheDocument();
  });

  it("renders typing state for a participant rail consumer", () => {
    render(
      <ParticipantPortraitRail
        participants={participantPortraitRailFixtures.typing}
        viewerParticipantId="p-rhea"
        ariaLabel="Side chat participants"
      />,
    );

    expect(screen.getByLabelText("Side chat participants")).toBeInTheDocument();
    expect(screen.getByLabelText("Bryn: typing")).toBeInTheDocument();
    expect(screen.getByLabelText("Bryn status: Typing")).toBeInTheDocument();
  });

  it("renders on-stage acting and revision states", () => {
    render(
      <ParticipantPortraitRail
        participants={participantPortraitRailFixtures.changesRequested}
        viewerParticipantId="p-rhea"
        ariaLabel="On-stage participants"
      />,
    );

    expect(screen.getByLabelText("On-stage participants")).toBeInTheDocument();
    expect(screen.getByLabelText("Rhea: changes requested")).toBeInTheDocument();
    expect(screen.getByLabelText("Rhea status: Changes requested")).toBeInTheDocument();
  });
});
