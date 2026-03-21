import { HUDNavbar } from "../hud-navbar/HUDNavbar";
import { SideChatPanel } from "../chat/side-chat-panel/SideChatPanel";
import type { PlayerHUDShellProps } from "./contract";

// PlayerHUDShell is the v2 composition viewport with a top navbar and tab
// content below. The side-chat tab renders the SideChatPanel; other tabs show
// a placeholder until their content panels are built.
export function PlayerHUDShell({
  activeTab,
  onTabChange,
  sideChat,
  sideChatDraft,
  onSideChatDraftChange,
  onSideChatSend,
}: PlayerHUDShellProps) {
  return (
    <main aria-label="Player HUD shell" className="flex h-dvh w-full flex-col">
      <HUDNavbar activeTab={activeTab} onTabChange={onTabChange} />

      <div className="flex min-h-0 flex-1">
        {activeTab === "side-chat" ? (
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
    </main>
  );
}
