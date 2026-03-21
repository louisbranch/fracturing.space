import { OnStageCompose } from "../on-stage-compose/OnStageCompose";
import { OnStageSceneCard } from "../on-stage-scene-card/OnStageSceneCard";
import { OnStageSlotList } from "../on-stage-slot-list/OnStageSlotList";
import { onStageStatusBadge } from "../../shared/view-models";
import type { OnStagePanelProps } from "./contract";

export function OnStagePanel({
  state,
  draft,
  onDraftChange,
  onSubmit,
  onSubmitAndYield,
  onYield,
  onUnyield,
  onCharacterInspect,
}: OnStagePanelProps) {
  const status = onStageStatusBadge(state);

  return (
    <section aria-label="On Stage" className="flex min-h-0 flex-1 flex-col">
      <div className="hud-panel-scroll-region">
        <OnStageSceneCard
          sceneName={state.sceneName}
          sceneDescription={state.sceneDescription}
          gmOutputText={state.gmOutputText}
          frameText={state.frameText}
          actingCharacterNames={state.actingCharacterNames}
          status={status}
        />
        <OnStageSlotList
          participants={state.participants}
          slots={state.slots}
          actingParticipantIds={state.actingParticipantIds}
          viewerParticipantId={state.viewerParticipantId}
          onCharacterInspect={onCharacterInspect}
        />
      </div>
      <OnStageCompose
        draft={draft}
        controls={state.viewerControls}
        mechanicsExtension={state.mechanicsExtension}
        onDraftChange={onDraftChange}
        onSubmit={onSubmit}
        onSubmitAndYield={onSubmitAndYield}
        onYield={onYield}
        onUnyield={onUnyield}
      />
    </section>
  );
}
