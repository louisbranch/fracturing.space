import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { onStageFixtureCatalog } from "./fixtures";
import { OnStageSlotCard } from "./OnStageSlotCard";

describe("OnStageSlotCard", () => {
  it("renders a committed participant-owned action slot", () => {
    const state = onStageFixtureCatalog.viewerPosted;
    const slot = state.slots[0];
    const participant = state.participants.find((entry) => entry.id === slot?.participantId);
    if (!slot || !participant) {
      throw new Error("expected viewer slot fixture");
    }

    render(<OnStageSlotCard slot={slot} participant={participant} isViewer />);

    expect(screen.getByLabelText("On-stage slot by Rhea")).toBeInTheDocument();
    expect(screen.getByText("Rhea")).toBeInTheDocument();
    expect(screen.getByText("You")).toBeInTheDocument();
    expect(screen.getByText("as Aria")).toBeInTheDocument();
    expect(screen.getByText("Open")).toBeInTheDocument();
    expect(screen.getByText("Aria hooks a pry tool into the seam and braces for the ward's recoil.")).toBeInTheDocument();
    expect(screen.getByText("16:31")).toBeInTheDocument();
  });

  it("renders the revision request when changes are requested", () => {
    const state = onStageFixtureCatalog.changesRequested;
    const slot = state.slots[0];
    const participant = state.participants.find((entry) => entry.id === slot?.participantId);
    if (!slot || !participant) {
      throw new Error("expected changes-requested fixture");
    }

    render(<OnStageSlotCard slot={slot} participant={participant} isViewer />);

    expect(screen.getByText("Changes Requested")).toBeInTheDocument();
    expect(screen.getByLabelText("Revision request")).toBeInTheDocument();
    expect(screen.getByText("Commit to how Aria keeps contact off the seam itself.")).toBeInTheDocument();
  });
});
