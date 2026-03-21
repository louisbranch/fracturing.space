import { BackstageContextCard } from "../backstage-context-card/BackstageContextCard";
import { BackstageCompose } from "../backstage-compose/BackstageCompose";
import { BackstageOOCList } from "../backstage-ooc-list/BackstageOOCList";
import { BackstageStatusBanner } from "../backstage-status-banner/BackstageStatusBanner";
import type { BackstagePanelProps } from "./contract";

export function BackstagePanel({
  state,
  draft,
  onDraftChange,
  onSend,
  onReadyToggle,
}: BackstagePanelProps) {
  const viewer = state.participants.find((participant) => participant.id === state.viewerParticipantId);

  return (
    <section aria-label="Backstage" className="flex min-h-0 flex-1 flex-col">
      <BackstageStatusBanner
        mode={state.mode}
        resumeState={state.resumeState}
        viewerReady={Boolean(viewer?.readyToResume)}
        onViewerReadyToggle={onReadyToggle}
      />
      {state.mode === "open" ? (
        <BackstageContextCard
          sceneName={state.sceneName}
          pausedPromptText={state.pausedPromptText}
          reason={state.reason}
        />
      ) : null}
      <BackstageOOCList
        messages={state.messages}
        participants={state.participants}
        viewerParticipantId={state.viewerParticipantId}
      />
      <BackstageCompose
        draft={draft}
        disabled={state.mode !== "open"}
        onDraftChange={onDraftChange}
        onSend={onSend}
      />
    </section>
  );
}
