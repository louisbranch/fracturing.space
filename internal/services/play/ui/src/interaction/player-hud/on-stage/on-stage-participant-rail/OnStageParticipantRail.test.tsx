import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { onStageFixtureCatalog } from "./fixtures";
import { OnStageParticipantRail } from "./OnStageParticipantRail";

describe("OnStageParticipantRail", () => {
  it("renders acting and yielded participant states for on-stage play", () => {
    render(
      <OnStageParticipantRail
        participants={onStageFixtureCatalog.yieldedWaiting.participants}
        viewerParticipantId={onStageFixtureCatalog.yieldedWaiting.viewerParticipantId}
      />,
    );

    expect(screen.getByLabelText("On-stage participants")).toBeInTheDocument();
    expect(screen.getByLabelText("Rhea: yielded")).toBeInTheDocument();
    expect(screen.getByLabelText("Bryn: active")).toBeInTheDocument();
    expect(screen.getByLabelText("Guide GM authority")).toBeInTheDocument();
  });
});
