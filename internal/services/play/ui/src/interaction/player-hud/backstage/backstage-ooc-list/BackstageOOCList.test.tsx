import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { BackstageOOCList } from "./BackstageOOCList";
import { backstageFixtureCatalog } from "./fixtures";

describe("BackstageOOCList", () => {
  it("renders an empty OOC state", () => {
    render(
      <BackstageOOCList
        messages={backstageFixtureCatalog.openEmpty.messages}
        participants={backstageFixtureCatalog.openEmpty.participants}
        viewerParticipantId={backstageFixtureCatalog.openEmpty.viewerParticipantId}
      />,
    );

    expect(screen.getByLabelText("Backstage OOC messages")).toBeInTheDocument();
    expect(screen.getByText("No OOC messages yet")).toBeInTheDocument();
  });

  it("renders active OOC discussion messages", () => {
    render(
      <BackstageOOCList
        messages={backstageFixtureCatalog.openDiscussion.messages}
        participants={backstageFixtureCatalog.openDiscussion.participants}
        viewerParticipantId={backstageFixtureCatalog.openDiscussion.viewerParticipantId}
      />,
    );

    expect(screen.getByText("Does the ward react to metal touching the seam or only skin?")).toBeInTheDocument();
    expect(screen.getByText("Guide")).toBeInTheDocument();
  });
});
