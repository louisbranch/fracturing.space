import { BackstagePanel } from "../backstage/backstage-panel/BackstagePanel";
import { BackstageParticipantRail } from "../backstage/backstage-participant-rail/BackstageParticipantRail";
import { HUDNavbar } from "../hud-navbar/HUDNavbar";
import { SideChatPanel } from "../chat/side-chat-panel/SideChatPanel";
import { SideChatParticipantRail } from "../chat/side-chat-participant-rail/SideChatParticipantRail";
import type { PlayerHUDShellProps } from "./contract";

// PlayerHUDShell is the player HUD composition viewport with a top navbar and
// tab content below. The shell also owns the shared right-side participant rail
// so all three top-level tabs keep the same overall layout.
export function PlayerHUDShell({
  activeTab,
  onTabChange,
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
  ) : (
    <BackstageParticipantRail
      participants={backstage.participants}
      viewerParticipantId={backstage.viewerParticipantId}
      gmAuthorityParticipantId={backstage.gmAuthorityParticipantId}
      ariaLabel={activeTab === "on-stage" ? "On-stage participants" : "Backstage participants"}
    />
  );

  return (
    <main aria-label="Player HUD shell" className="flex h-dvh w-full flex-col">
      <HUDNavbar activeTab={activeTab} onTabChange={onTabChange} />

      <div className="flex min-h-0 flex-1">
        <div className="flex min-h-0 flex-1">
          {activeTab === "backstage" ? (
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
            <div className="m-4 flex flex-1 items-center justify-center rounded-box border border-dashed border-base-300/70 bg-base-200/25">
              <span className="text-sm text-base-content/50">
                {activeTab} — content coming soon
              </span>
            </div>
          )}
        </div>
        {participantRail}
      </div>
    </main>
  );
}
