import { BackstagePanel } from "../backstage/backstage-panel/BackstagePanel";
import { BackstageParticipantRail } from "../backstage/backstage-participant-rail/BackstageParticipantRail";
import { HUDNavbar } from "../hud-navbar/HUDNavbar";
import { SideChatPanel } from "../chat/side-chat-panel/SideChatPanel";
import { SideChatParticipantRail } from "../chat/side-chat-participant-rail/SideChatParticipantRail";
import { OnStagePanel } from "../on-stage/on-stage-panel/OnStagePanel";
import { OnStageParticipantRail } from "../on-stage/on-stage-participant-rail/OnStageParticipantRail";
import type { PlayerHUDShellProps } from "./contract";

// PlayerHUDShell is the player HUD composition viewport with a top navbar and
// tab content below. The shell also owns the shared right-side participant rail
// so all three top-level tabs keep the same overall layout.
export function PlayerHUDShell({
  activeTab,
  onTabChange,
  onStage,
  onStageDraft,
  onOnStageDraftChange,
  onOnStageSubmit,
  onOnStageSubmitAndYield,
  onOnStageYield,
  onOnStageUnyield,
  backstage,
  backstageDraft,
  onBackstageDraftChange,
  onBackstageSend,
  onBackstageReadyToggle,
  sideChat,
  sideChatDraft,
  onSideChatDraftChange,
  onSideChatSend,
}: PlayerHUDShellProps) {
  const participantRail = activeTab === "side-chat" ? (
    <SideChatParticipantRail
      participants={sideChat.participants}
      viewerParticipantId={sideChat.viewerParticipantId}
    />
  ) : activeTab === "on-stage" ? (
    <OnStageParticipantRail
      participants={onStage.participants}
      viewerParticipantId={onStage.viewerParticipantId}
    />
  ) : (
    <BackstageParticipantRail
      participants={backstage.participants}
      viewerParticipantId={backstage.viewerParticipantId}
      gmAuthorityParticipantId={backstage.gmAuthorityParticipantId}
      ariaLabel="Backstage participants"
    />
  );

  return (
    <main aria-label="Player HUD shell" className="play-density-hud flex h-dvh w-full flex-col">
      <HUDNavbar activeTab={activeTab} onTabChange={onTabChange} />

      <div className="flex min-h-0 flex-1">
        <div className="flex min-h-0 flex-1">
          {activeTab === "on-stage" ? (
            <OnStagePanel
              state={onStage}
              draft={onStageDraft}
              onDraftChange={onOnStageDraftChange}
              onSubmit={onOnStageSubmit}
              onSubmitAndYield={onOnStageSubmitAndYield}
              onYield={onOnStageYield}
              onUnyield={onOnStageUnyield}
            />
          ) : activeTab === "backstage" ? (
            <BackstagePanel
              state={backstage}
              draft={backstageDraft}
              onDraftChange={onBackstageDraftChange}
              onSend={onBackstageSend}
              onReadyToggle={onBackstageReadyToggle}
            />
          ) : activeTab === "side-chat" ? (
            <SideChatPanel
              state={sideChat}
              draft={sideChatDraft}
              onDraftChange={onSideChatDraftChange}
              onSend={onSideChatSend}
            />
          ) : (
            <div />
          )}
        </div>
        {participantRail}
      </div>
    </main>
  );
}
