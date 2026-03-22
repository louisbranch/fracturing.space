import { useCallback, useEffect, useRef, useState } from "react";
import { useTransitionPreferences } from "./TransitionPreferencesContext";

type TransitionEffectsResult = {
  sceneTransitionKey: number;
  interactionTransitionActive: boolean;
  clearInteractionTransition: () => void;
};

// useTransitionEffects detects scene and interaction ID changes and returns
// signals that drive visual overlays and audio playback. On initial mount the
// refs are undefined so no effects fire.
export function useTransitionEffects(
  sceneId: string | undefined,
  interactionId: string | undefined,
): TransitionEffectsResult {
  const { preferences } = useTransitionPreferences();
  const prevSceneIdRef = useRef<string | undefined>(undefined);
  const prevInteractionIdRef = useRef<string | undefined>(undefined);
  const [sceneTransitionKey, setSceneTransitionKey] = useState(0);
  const [interactionTransitionActive, setInteractionTransitionActive] = useState(false);

  const clearInteractionTransition = useCallback(() => {
    setInteractionTransitionActive(false);
  }, []);

  useEffect(() => {
    // Initial mount — capture current IDs without firing effects.
    if (prevSceneIdRef.current === undefined) {
      prevSceneIdRef.current = sceneId;
      prevInteractionIdRef.current = interactionId;
      return;
    }

    const sceneChanged = sceneId !== prevSceneIdRef.current;
    const interactionChanged = interactionId !== prevInteractionIdRef.current;

    prevSceneIdRef.current = sceneId;
    prevInteractionIdRef.current = interactionId;

    if (sceneChanged) {
      // Scene change is the superset — fire scene transition, skip interaction.
      if (preferences.sceneVisual) {
        setSceneTransitionKey((k) => k + 1);
      }
      setInteractionTransitionActive(false);
      return;
    }

    if (interactionChanged) {
      if (preferences.interactionVisual) {
        setInteractionTransitionActive(true);
      }
    }
  }, [sceneId, interactionId, preferences.sceneVisual, preferences.interactionVisual]);

  return { sceneTransitionKey, interactionTransitionActive, clearInteractionTransition };
}
