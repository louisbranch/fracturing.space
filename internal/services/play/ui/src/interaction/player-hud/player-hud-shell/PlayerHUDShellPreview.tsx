import { useState } from "react";
import type { BackstageParticipant } from "../backstage/shared/contract";
import type { HUDNavbarTab, PlayerHUDState } from "../shared/contract";
import { PlayerHUDShell } from "./PlayerHUDShell";

type PlayerHUDShellPreviewProps = {
  initialState: PlayerHUDState;
};

export function PlayerHUDShellPreview({ initialState }: PlayerHUDShellPreviewProps) {
  const [activeTab, setActiveTab] = useState<HUDNavbarTab>(initialState.activeTab);
  const [backstage, setBackstage] = useState(initialState.backstage);
  const [backstageDraft, setBackstageDraft] = useState("");
  const [draft, setDraft] = useState("");

  function nextResumeState(
    mode: PlayerHUDState["backstage"]["mode"],
    participants: BackstageParticipant[],
  ): PlayerHUDState["backstage"]["resumeState"] {
    if (mode !== "open") {
      return "inactive";
    }
    const players = participants.filter((participant) => participant.role === "player");
    return players.length > 0 && players.every((participant) => participant.readyToResume)
      ? "waiting-on-gm"
      : "collecting-ready";
  }

  function handleBackstageReadyToggle() {
    setBackstage((current) => {
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

  function handleBackstageSend() {
    const body = backstageDraft.trim();
    if (backstage.mode !== "open" || body.length === 0) {
      return;
    }
    setBackstage((current) => ({
      ...current,
      messages: [
        ...current.messages,
        {
          id: `shell-preview-${current.messages.length + 1}`,
          participantId: current.viewerParticipantId,
          body,
          sentAt: new Date().toISOString(),
        },
      ],
    }));
    setBackstageDraft("");
  }

  return (
    <PlayerHUDShell
      activeTab={activeTab}
      onTabChange={setActiveTab}
      backstage={backstage}
      backstageDraft={backstageDraft}
      onBackstageDraftChange={setBackstageDraft}
      onBackstageSend={handleBackstageSend}
      onBackstageReadyToggle={handleBackstageReadyToggle}
      sideChat={initialState.sideChat}
      sideChatDraft={draft}
      onSideChatDraftChange={setDraft}
      onSideChatSend={() => setDraft("")}
    />
  );
}
