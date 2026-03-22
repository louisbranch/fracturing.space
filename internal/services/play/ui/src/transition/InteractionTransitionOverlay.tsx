import { useCallback, useEffect, useRef } from "react";

type InteractionTransitionOverlayProps = {
  active: boolean;
  onTransitionEnd?: () => void;
  fallbackMs?: number;
  children: React.ReactNode;
};

// InteractionTransitionOverlay wraps children with the interaction-pulse CSS
// animation. A fallback timer ensures the active state clears even if
// onAnimationEnd doesn't fire, mirroring the safety pattern used by
// SceneTransitionOverlay.
export function InteractionTransitionOverlay({
  active,
  onTransitionEnd,
  fallbackMs = 2100,
  children,
}: InteractionTransitionOverlayProps) {
  const fallbackTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const clearFallback = useCallback(() => {
    if (fallbackTimerRef.current) {
      clearTimeout(fallbackTimerRef.current);
      fallbackTimerRef.current = null;
    }
  }, []);

  useEffect(() => {
    if (!active) return;

    fallbackTimerRef.current = setTimeout(() => {
      fallbackTimerRef.current = null;
      onTransitionEnd?.();
    }, fallbackMs);

    return clearFallback;
  }, [active, onTransitionEnd, fallbackMs, clearFallback]);

  const handleAnimationEnd = useCallback(() => {
    clearFallback();
    onTransitionEnd?.();
  }, [clearFallback, onTransitionEnd]);

  return (
    <div
      className={active ? "interaction-pulse" : undefined}
      onAnimationEnd={active ? handleAnimationEnd : undefined}
    >
      {children}
    </div>
  );
}
