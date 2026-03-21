import { useState } from "react";
import type { OnStageState } from "../shared/contract";
import { OnStagePanel } from "./OnStagePanel";

type OnStagePanelPreviewProps = {
  initialState: OnStageState;
};

function updateViewerState(
  current: OnStageState,
  yielded: boolean,
  nextMode: OnStageState["mode"],
): OnStageState {
  const participants = current.participants.map((participant) =>
    participant.id === current.viewerParticipantId
      ? {
          ...participant,
          railStatus:
            nextMode === "changes-requested"
              ? "changes-requested"
              : yielded
                ? "yielded"
                : current.mode === "acting"
                  ? "active"
                  : participant.railStatus,
        }
      : participant,
  );

  return {
    ...current,
    mode: nextMode,
    participants,
    viewerControls: {
      ...current.viewerControls,
      canSubmit: !yielded && nextMode !== "ooc-blocked" && nextMode !== "waiting-on-gm",
      canSubmitAndYield: !yielded && nextMode !== "ooc-blocked" && nextMode !== "waiting-on-gm",
      canYield: !yielded && nextMode === "acting",
      canUnyield: yielded && nextMode === "yielded-waiting",
      disabledReason: yielded
        ? "You have already yielded. Unyield if you need to revise before the beat closes."
        : current.viewerControls.disabledReason,
    },
  };
}

export function OnStagePanelPreview({ initialState }: OnStagePanelPreviewProps) {
  const initialViewerSlot =
    initialState.slots.find((slot) => slot.participantId === initialState.viewerParticipantId)?.body ?? "";
  const [state, setState] = useState(initialState);
  const [draft, setDraft] = useState(initialViewerSlot);

  function writeViewerSlot(yielded: boolean, nextMode: OnStageState["mode"]) {
    const body = draft.trim();
    if (body.length === 0 && !yielded) {
      return;
    }

    setState((current) => {
      const nextSlots = [...current.slots];
      const index = nextSlots.findIndex((slot) => slot.participantId === current.viewerParticipantId);
      const currentSlot = index >= 0 ? nextSlots[index] : undefined;
      const slot = {
        id: currentSlot?.id ?? `${current.viewerParticipantId}-preview`,
        participantId: current.viewerParticipantId,
        characters:
          currentSlot?.characters.length
            ? currentSlot.characters
            : current.participants.find((participant) => participant.id === current.viewerParticipantId)?.characters ?? [],
        body: body || currentSlot?.body,
        updatedAt: new Date().toISOString(),
        yielded,
        reviewState: yielded ? "under-review" : "open",
      } as const;

      if (index >= 0) {
        nextSlots[index] = slot;
      } else {
        nextSlots.unshift(slot);
      }

      return {
        ...updateViewerState(current, yielded, nextMode),
        slots: nextSlots,
      };
    });
  }

  function handleSubmit() {
    writeViewerSlot(false, "acting");
  }

  function handleSubmitAndYield() {
    writeViewerSlot(true, "yielded-waiting");
  }

  function handleYield() {
    writeViewerSlot(true, "yielded-waiting");
  }

  function handleUnyield() {
    setState((current) => ({
      ...updateViewerState(current, false, "acting"),
      slots: current.slots.map((slot) =>
        slot.participantId === current.viewerParticipantId
          ? { ...slot, yielded: false, reviewState: "open" }
          : slot,
      ),
    }));
  }

  return (
    <OnStagePanel
      state={state}
      draft={draft}
      onDraftChange={setDraft}
      onSubmit={handleSubmit}
      onSubmitAndYield={handleSubmitAndYield}
      onYield={handleYield}
      onUnyield={handleUnyield}
    />
  );
}
