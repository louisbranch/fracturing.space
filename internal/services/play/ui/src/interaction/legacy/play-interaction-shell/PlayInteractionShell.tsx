import { ActingSetRail } from "../acting-set-rail/ActingSetRail";
import { AITurnStatusBanner } from "../ai-turn-status-banner/AITurnStatusBanner";
import { CharacterReferenceRail } from "../character-reference-rail/CharacterReferenceRail";
import { ChatSidecar } from "../chat-sidecar/ChatSidecar";
import { GMReviewPanel } from "../gm-review-panel/GMReviewPanel";
import { OOCOverlayPanel } from "../ooc-overlay-panel/OOCOverlayPanel";
import { PlayerSlotBoard } from "../player-slot-board/PlayerSlotBoard";
import { SceneFramePanel } from "../scene-frame-panel/SceneFramePanel";
import type { PlayInteractionShellProps } from "./contract";

// PlayInteractionShell is the composition-only Storybook surface that proves
// the isolated interaction slices fit together before any runtime wiring exists.
export function PlayInteractionShell({ state, references, showChatSidecar = true }: PlayInteractionShellProps) {
  return (
    <main className="preview-shell">
      <header className="preview-panel">
        <div className="preview-panel-body gap-3">
          <span className="preview-kicker">Play Interaction Shell</span>
          <div className="flex flex-wrap items-end justify-between gap-3">
            <div>
              <h1 className="font-display text-4xl text-base-content">{state.campaignName}</h1>
              <p className="mt-2 text-sm leading-6 text-base-content/70">
                {state.sessionName} · {state.systemName} · viewing as {state.viewerName}
              </p>
            </div>
            <span className="badge badge-outline uppercase">{state.viewerRole}</span>
          </div>
        </div>
      </header>

      <div className={`grid gap-6 ${showChatSidecar ? "xl:grid-cols-[minmax(0,1fr)_22rem]" : ""}`}>
        <div className="space-y-6">
          {state.aiTurn && state.aiTurn.status !== "idle" ? <AITurnStatusBanner aiTurn={state.aiTurn} /> : null}
          <SceneFramePanel phase={state.phase} scene={state.scene} />
          {state.phase.oocOpen && state.ooc ? <OOCOverlayPanel ooc={state.ooc} phase={state.phase} /> : null}
          <ActingSetRail actingSet={state.actingSet} />
          <PlayerSlotBoard slots={state.slots} />
          {state.phase.status === "gm_review" ? <GMReviewPanel slots={state.slots} /> : null}
          <CharacterReferenceRail {...references} />
        </div>

        {showChatSidecar ? <ChatSidecar messages={state.chat} /> : null}
      </div>
    </main>
  );
}
