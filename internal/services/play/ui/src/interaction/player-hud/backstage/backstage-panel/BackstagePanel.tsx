import { BackstageContextCard } from "../backstage-context-card/BackstageContextCard";
import { BackstageCompose } from "../backstage-compose/BackstageCompose";
import { BackstageOOCList } from "../backstage-ooc-list/BackstageOOCList";
import { backstageStatusBadge } from "../../shared/view-models";
import type { BackstagePanelProps } from "./contract";

export function BackstagePanel({
  state,
  draft,
  onDraftChange,
  onSend,
  onReadyToggle,
}: BackstagePanelProps) {
  const viewer = state.participants.find((participant) => participant.id === state.viewerParticipantId);
  const viewerReady = Boolean(viewer?.readyToResume);
  const status = backstageStatusBadge(state);

  return (
    <section aria-label="Backstage" className="flex min-h-0 flex-1 flex-col">
      <div className="hud-panel-scroll-region">
        <BackstageContextCard
          sceneName={state.sceneName}
          pausedPromptText={state.pausedPromptText}
          reason={state.reason}
          status={status}
        />
        <BackstageOOCList
          messages={state.messages}
          participants={state.participants}
          viewerParticipantId={state.viewerParticipantId}
        />
      </div>
      <BackstageCompose
        draft={draft}
        viewerReady={viewerReady}
        disabled={state.mode !== "open"}
        onDraftChange={onDraftChange}
        onSend={onSend}
        onReadyToggle={onReadyToggle}
      />
    </section>
  );
}
