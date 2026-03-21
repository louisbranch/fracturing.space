import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { BackstageParticipantRail } from "./BackstageParticipantRail";
import { backstageFixtureCatalog } from "./fixtures";

describe("BackstageParticipantRail", () => {
  it("renders the participant rail with portrait state labels", () => {
    render(
      <BackstageParticipantRail
        participants={backstageFixtureCatalog.waitingOnGM.participants}
        viewerParticipantId={backstageFixtureCatalog.waitingOnGM.viewerParticipantId}
        gmAuthorityParticipantId={backstageFixtureCatalog.waitingOnGM.gmAuthorityParticipantId}
      />,
    );

    expect(screen.getByLabelText("Backstage participants")).toBeInTheDocument();
    expect(screen.getByLabelText("Rhea: ready")).toBeInTheDocument();
    expect(screen.getByLabelText("Rhea status: Ready to resume")).toBeInTheDocument();
    expect(screen.getByLabelText("Guide: waiting")).toBeInTheDocument();
    expect(screen.getByLabelText("Guide GM authority")).toBeInTheDocument();
  });

  it("shows typing status in preference to ready state", () => {
    render(
      <BackstageParticipantRail
        participants={backstageFixtureCatalog.openDiscussion.participants}
        viewerParticipantId={backstageFixtureCatalog.openDiscussion.viewerParticipantId}
        gmAuthorityParticipantId={backstageFixtureCatalog.openDiscussion.gmAuthorityParticipantId}
      />,
    );

    expect(screen.getByLabelText("Bryn: typing")).toBeInTheDocument();
    expect(screen.getByLabelText("Bryn status: Typing")).toBeInTheDocument();
  });
});
