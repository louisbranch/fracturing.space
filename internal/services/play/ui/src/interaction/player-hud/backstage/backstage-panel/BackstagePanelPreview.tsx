import { useState } from "react";
import type { BackstageParticipant, BackstageState } from "../shared/contract";
import { BackstagePanel } from "./BackstagePanel";

type BackstagePanelPreviewProps = {
  initialState: BackstageState;
};

function nextResumeState(mode: BackstageState["mode"], participants: BackstageParticipant[]): BackstageState["resumeState"] {
  if (mode !== "open") {
    return "inactive";
  }
  const players = participants.filter((participant) => participant.role === "player");
  return players.length > 0 && players.every((participant) => participant.readyToResume)
    ? "waiting-on-gm"
    : "collecting-ready";
}

export function BackstagePanelPreview({ initialState }: BackstagePanelPreviewProps) {
  const [state, setState] = useState(initialState);
  const [draft, setDraft] = useState("");

  function handleReadyToggle() {
    setState((current) => {
      const participants = current.participants.map((participant) =>
        participant.id === current.viewerParticipantId
          ? { ...participant, readyToResume: !participant.readyToResume }
          : participant,
      );
      return {
        ...current,
        participants,
        resumeState: nextResumeState(current.mode, participants),
      };
    });
  }

  function handleSend() {
    const body = draft.trim();
    if (state.mode !== "open" || body.length === 0) {
      return;
    }
    setState((current) => ({
      ...current,
      messages: [
        ...current.messages,
        {
          id: `preview-${current.messages.length + 1}`,
          participantId: current.viewerParticipantId,
          body,
          sentAt: new Date().toISOString(),
        },
      ],
    }));
    setDraft("");
  }

  return (
    <BackstagePanel
      state={state}
      draft={draft}
      onDraftChange={setDraft}
      onSend={handleSend}
      onReadyToggle={handleReadyToggle}
    />
  );
}
