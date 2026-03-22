import { useEffect, useRef, useState } from "react";
import { OnStageCompose } from "../on-stage-compose/OnStageCompose";
import { OnStageGMInteractionCard } from "../on-stage-gm-interaction-card/OnStageGMInteractionCard";
import { OnStageSceneCard } from "../on-stage-scene-card/OnStageSceneCard";
import { OnStageSlotList } from "../on-stage-slot-list/OnStageSlotList";
import { onStageStatusBadge } from "../../shared/view-models";
import type { OnStagePanelProps } from "./contract";

export function OnStagePanel({
  state,
  draft,
  interactionTransitionActive,
  onInteractionTransitionEnd,
  onDraftChange,
  onSubmit,
  onSubmitAndYield,
  onYield,
  onUnyield,
  onCharacterInspect,
}: OnStagePanelProps) {
  const status = onStageStatusBadge(state);
  const [descriptionExpanded, setDescriptionExpanded] = useState(
    state.scene.resolvedInteractionCount < 1,
  );
  const [descriptionTouched, setDescriptionTouched] = useState(false);
  const previousSceneId = useRef(state.scene.id);
  const previousResolvedInteractionCount = useRef(state.scene.resolvedInteractionCount);

  useEffect(() => {
    if (state.scene.id !== previousSceneId.current) {
      previousSceneId.current = state.scene.id;
      previousResolvedInteractionCount.current = state.scene.resolvedInteractionCount;
      setDescriptionExpanded(state.scene.resolvedInteractionCount < 1);
      setDescriptionTouched(false);
      return;
    }

    if (
      previousResolvedInteractionCount.current < 1
      && state.scene.resolvedInteractionCount >= 1
      && !descriptionTouched
    ) {
      setDescriptionExpanded(false);
    }

    previousResolvedInteractionCount.current = state.scene.resolvedInteractionCount;
  }, [descriptionTouched, state.scene.id, state.scene.resolvedInteractionCount]);

  return (
    <section aria-label="On Stage" className="flex min-h-0 min-w-0 flex-1 flex-col">
      <div className="hud-panel-scroll-region min-w-0">
        <OnStageSceneCard
          sceneName={state.scene.name}
          sceneDescription={state.scene.description}
          sceneCharacters={state.scene.characters}
          resolvedInteractionCount={state.scene.resolvedInteractionCount}
          expanded={descriptionExpanded}
          onToggle={() => {
            setDescriptionTouched(true);
            setDescriptionExpanded((current) => !current);
          }}
          onCharacterInspect={(characterId) => {
            const participant = state.participants.find((entry) =>
              entry.characters.some((character) => character.id === characterId),
            );
            if (!participant) {
              return;
            }
            onCharacterInspect?.(participant.id, characterId);
          }}
        />
        <OnStageGMInteractionCard
          currentInteraction={state.currentInteraction}
          interactionHistory={state.interactionHistory}
          currentStatus={status}
          transitionActive={interactionTransitionActive}
          onTransitionEnd={onInteractionTransitionEnd}
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
