import { AIDebugPanel } from "../ai-debug/ai-debug-panel/AIDebugPanel";
import { BackstagePanel } from "../backstage/backstage-panel/BackstagePanel";
import { BackstageParticipantRail } from "../backstage/backstage-participant-rail/BackstageParticipantRail";
import { SideChatPanel } from "../chat/side-chat-panel/SideChatPanel";
import { SideChatParticipantRail } from "../chat/side-chat-participant-rail/SideChatParticipantRail";
import { OnStagePanel } from "../on-stage/on-stage-panel/OnStagePanel";
import { OnStageParticipantRail } from "../on-stage/on-stage-participant-rail/OnStageParticipantRail";
import { HUDNavbar } from "../shared/hud-navbar/HUDNavbar";
import { PanelErrorBoundary } from "../shared/PanelErrorBoundary";
import { PlayerHUDDrawerSidebar } from "../shared/player-hud-drawer-sidebar/PlayerHUDDrawerSidebar";
import type { PlayerHUDShellProps } from "./contract";

// PlayerHUDShell is the player HUD composition viewport with a top navbar and
// tab content below. The shell also owns the shared right-side participant rail
// so all three top-level tabs keep the same overall layout.
export function PlayerHUDShell({
  activeTab,
  aiDebugEnabled,
  connectionState,
  campaignNavigation,
  isSidebarOpen,
  onSidebarOpenChange,
  onTabChange,
  onSettingsOpen,
  interactionTransitionActive,
  onInteractionTransitionEnd,
  onStage,
  onStageDraft,
  onOnStageDraftChange,
  onOnStageSubmit,
  onOnStageSubmitAndYield,
  onOnStageYield,
  onOnStageUnyield,
  onCharacterInspect,
  onParticipantInspect,
  backstage,
  backstageDraft,
  onBackstageDraftChange,
  onBackstageSend,
  onBackstageReadyToggle,
  sideChat,
  sideChatDraft,
  onSideChatDraftChange,
  onSideChatSend,
  aiDebug,
  onAIDebugLoadMore,
  onAIDebugToggleTurn,
}: PlayerHUDShellProps) {
  const participantRail = activeTab === "ai-debug" ? null : activeTab === "side-chat" ? (
    <SideChatParticipantRail
      participants={sideChat.participants}
      viewerParticipantId={sideChat.viewerParticipantId}
      aiOwnerParticipantId={onStage.aiOwnerParticipantId}
      aiStatus={onStage.aiStatus}
      onParticipantInspect={onParticipantInspect}
    />
  ) : activeTab === "on-stage" ? (
    <OnStageParticipantRail
      participants={onStage.participants}
      viewerParticipantId={onStage.viewerParticipantId}
      aiOwnerParticipantId={onStage.aiOwnerParticipantId}
      aiStatus={onStage.aiStatus}
      onParticipantInspect={onParticipantInspect}
    />
  ) : (
    <BackstageParticipantRail
      participants={backstage.participants}
      viewerParticipantId={backstage.viewerParticipantId}
      gmAuthorityParticipantId={backstage.gmAuthorityParticipantId}
      aiOwnerParticipantId={onStage.aiOwnerParticipantId}
      aiStatus={onStage.aiStatus}
      ariaLabel="Backstage participants"
      onParticipantInspect={onParticipantInspect}
    />
  );

  return (
    <div className={`drawer h-dvh ${isSidebarOpen ? "drawer-open" : ""}`}>
      <input
        type="checkbox"
        className="drawer-toggle"
        checked={isSidebarOpen}
        aria-label="Player HUD sidebar toggle"
        onChange={(event) => onSidebarOpenChange(event.currentTarget.checked)}
      />
      <div className="drawer-content">
        <main aria-label="Player HUD shell" className="play-density-hud flex h-dvh w-full flex-col bg-base-300">
          <HUDNavbar
            activeTab={activeTab}
            aiDebugEnabled={aiDebugEnabled}
            connectionState={connectionState}
            isSidebarOpen={isSidebarOpen}
            onSidebarOpenChange={onSidebarOpenChange}
            onTabChange={onTabChange}
          />

          <div className="flex min-h-0 min-w-0 flex-1">
            <div className="flex min-h-0 min-w-0 flex-1">
              {activeTab === "on-stage" ? (
                <PanelErrorBoundary panelName="On Stage">
                  <OnStagePanel
                    state={onStage}
                    draft={onStageDraft}
                    interactionTransitionActive={interactionTransitionActive}
                    onInteractionTransitionEnd={onInteractionTransitionEnd}
                    onDraftChange={onOnStageDraftChange}
                    onSubmit={onOnStageSubmit}
                    onSubmitAndYield={onOnStageSubmitAndYield}
                    onYield={onOnStageYield}
                    onUnyield={onOnStageUnyield}
                    onCharacterInspect={onCharacterInspect}
                  />
                </PanelErrorBoundary>
              ) : activeTab === "backstage" ? (
                <PanelErrorBoundary panelName="Backstage">
                  <BackstagePanel
                    state={backstage}
                    draft={backstageDraft}
                    onDraftChange={onBackstageDraftChange}
                    onSend={onBackstageSend}
                    onReadyToggle={onBackstageReadyToggle}
                  />
                </PanelErrorBoundary>
              ) : activeTab === "side-chat" ? (
                <PanelErrorBoundary panelName="Side Chat">
                  <SideChatPanel
                    state={sideChat}
                    draft={sideChatDraft}
                    onDraftChange={onSideChatDraftChange}
                    onSend={onSideChatSend}
                  />
                </PanelErrorBoundary>
              ) : activeTab === "ai-debug" ? (
                <PanelErrorBoundary panelName="AI Debug">
                  <AIDebugPanel
                    state={aiDebug ?? { phase: "idle", turns: [], detailsByTurnId: {} }}
                    onLoadMore={onAIDebugLoadMore}
                    onToggleTurn={onAIDebugToggleTurn}
                  />
                </PanelErrorBoundary>
              ) : (
                <div />
              )}
            </div>
            {participantRail}
          </div>
        </main>
      </div>
      <div className="drawer-side z-20">
        <label
          aria-label="Close campaign sidebar"
          className="drawer-overlay"
          onClick={() => onSidebarOpenChange(false)}
        />
        <PlayerHUDDrawerSidebar
          navigation={campaignNavigation}
          onCharacterInspect={onCharacterInspect}
          onSettingsOpen={onSettingsOpen}
          onClose={() => onSidebarOpenChange(false)}
        />
      </div>
    </div>
  );
}
