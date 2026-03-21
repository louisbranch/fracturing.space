import { useState } from "react";
import type { HUDNavbarTab, PlayerHUDState } from "../shared/contract";
import { PlayerHUDShell } from "./PlayerHUDShell";

type PlayerHUDShellPreviewProps = {
  initialState: PlayerHUDState;
};

export function PlayerHUDShellPreview({ initialState }: PlayerHUDShellPreviewProps) {
  const [activeTab, setActiveTab] = useState<HUDNavbarTab>(initialState.activeTab);
  const [draft, setDraft] = useState("");

  return (
    <PlayerHUDShell
      activeTab={activeTab}
      onTabChange={setActiveTab}
      sideChat={initialState.sideChat}
      sideChatDraft={draft}
      onSideChatDraftChange={setDraft}
      onSideChatSend={() => setDraft("")}
    />
  );
}
