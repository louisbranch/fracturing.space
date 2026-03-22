import { useCallback, useRef } from "react";

type TransitionAudioOptions = {
  sceneUrl?: string;
  interactionUrl?: string;
};

// getOrCreateAudio lazily creates an Audio element on first call (which is
// always inside a user-gesture handler), avoiding browser autoplay warnings
// that fire when Audio elements are constructed during mount/effects.
function getOrCreateAudio(
  ref: React.RefObject<HTMLAudioElement | null>,
  url: string | undefined,
): HTMLAudioElement | null {
  if (!url) return null;
  if (ref.current) {
    if (ref.current.src === url) return ref.current;
    // URL changed — discard old element.
    ref.current.pause();
  }
  const audio = new Audio(url);
  audio.preload = "auto";
  ref.current = audio;
  return audio;
}

// useTransitionAudio provides play callbacks for scene and interaction SFX.
// Audio elements are created lazily on the first user-initiated play call so
// the browser never sees an autoplay attempt outside a gesture context.
export function useTransitionAudio({ sceneUrl, interactionUrl }: TransitionAudioOptions) {
  const sceneAudioRef = useRef<HTMLAudioElement | null>(null);
  const interactionAudioRef = useRef<HTMLAudioElement | null>(null);

  const playSceneSound = useCallback(() => {
    const audio = getOrCreateAudio(sceneAudioRef, sceneUrl);
    if (!audio) return;
    audio.currentTime = 0;
    audio.play().catch(() => {});
  }, [sceneUrl]);

  const playInteractionSound = useCallback(() => {
    const audio = getOrCreateAudio(interactionAudioRef, interactionUrl);
    if (!audio) return;
    audio.currentTime = 0;
    audio.play().catch(() => {});
  }, [interactionUrl]);

  return { playSceneSound, playInteractionSound };
}
