import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { onStageFixtureCatalog } from "./fixtures";
import { OnStageSlotList } from "./OnStageSlotList";

describe("OnStageSlotList", () => {
  it("renders one slot per acting participant, including empty acting slots", () => {
    render(
      <OnStageSlotList
        participants={onStageFixtureCatalog.actingEmpty.participants}
        slots={onStageFixtureCatalog.actingEmpty.slots}
        actingParticipantIds={onStageFixtureCatalog.actingEmpty.actingParticipantIds}
        viewerParticipantId={onStageFixtureCatalog.actingEmpty.viewerParticipantId}
      />,
    );

    expect(screen.getByLabelText("On-stage messages")).toBeInTheDocument();
    expect(screen.getByLabelText("On-stage slot by Rhea")).toBeInTheDocument();
    expect(screen.getByLabelText("On-stage slot by Bryn")).toBeInTheDocument();
    expect(screen.getByText("No committed post yet.")).toBeInTheDocument();
  });
});
