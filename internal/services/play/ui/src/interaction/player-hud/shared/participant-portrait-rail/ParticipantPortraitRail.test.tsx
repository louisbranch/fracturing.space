import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
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

    expect(screen.getByLabelText("Backstage participants")).toHaveClass("bg-base-300");
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
    const typingStatus = screen.getByLabelText("Bryn status: Typing");
    expect(typingStatus).toBeInTheDocument();
    expect(within(typingStatus).getByText("", { selector: ".loading.loading-dots" })).toBeInTheDocument();
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

  it("renders AI thinking on the GM portrait without hiding GM authority", () => {
    render(
      <ParticipantPortraitRail
        participants={[
          {
            id: "p-rhea",
            name: "Rhea",
            roleLabel: "PLAYER",
            characters: [],
            status: "idle",
          },
          {
            id: "p-guide",
            name: "Guide",
            roleLabel: "GM",
            characters: [],
            status: "idle",
            aiStatus: "thinking",
            ownsGMAuthority: true,
          },
        ]}
        viewerParticipantId="p-rhea"
        ariaLabel="AI participants"
      />,
    );

    expect(screen.getByLabelText("Guide: AI thinking")).toBeInTheDocument();
    expect(screen.getByLabelText("Guide status: AI thinking")).toBeInTheDocument();
    expect(screen.getByLabelText("Guide GM authority")).toBeInTheDocument();
  });

  it("renders AI failure on the GM portrait", () => {
    render(
      <ParticipantPortraitRail
        participants={[
          {
            id: "p-rhea",
            name: "Rhea",
            roleLabel: "PLAYER",
            characters: [],
            status: "idle",
          },
          {
            id: "p-guide",
            name: "Guide",
            roleLabel: "GM",
            characters: [],
            status: "idle",
            aiStatus: "failed",
            ownsGMAuthority: true,
          },
        ]}
        viewerParticipantId="p-rhea"
        ariaLabel="AI participants"
      />,
    );

    expect(screen.getByLabelText("Guide: AI failed")).toBeInTheDocument();
    expect(screen.getByLabelText("Guide status: AI failed")).toBeInTheDocument();
  });

  it("emits participant inspection clicks when portraits are interactive", async () => {
    const user = userEvent.setup();
    const onParticipantInspect = vi.fn();

    render(
      <ParticipantPortraitRail
        participants={participantPortraitRailFixtures.active}
        viewerParticipantId="p-rhea"
        ariaLabel="On-stage participants"
        onParticipantInspect={onParticipantInspect}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Inspect Bryn" }));
    expect(onParticipantInspect).toHaveBeenCalledWith("p-bryn");
  });

  it("orders the viewer first, then players, then the GM in the shared rail", () => {
    render(
      <ParticipantPortraitRail
        participants={[
          {
            id: "p-guide",
            name: "Guide",
            roleLabel: "GM",
            characters: [],
            status: "idle",
          },
          {
            id: "p-bryn",
            name: "Bryn",
            roleLabel: "PLAYER",
            characters: [],
            status: "typing",
          },
          {
            id: "p-rhea",
            name: "Rhea",
            roleLabel: "PLAYER",
            characters: [],
            status: "ready",
          },
        ]}
        viewerParticipantId="p-rhea"
        ariaLabel="Ordered participants"
      />,
    );

    const rail = screen.getByLabelText("Ordered participants");
    const orderedLabels = Array.from(
      rail.querySelectorAll(".font-medium.text-base-content"),
    ).map((element) => element.textContent);

    expect(orderedLabels).toEqual([
      "Rhea",
      "Bryn",
      "Guide",
    ]);
  });
});
