import { useCallback, useEffect, useRef, useState } from "react";

type SceneTransitionOverlayProps = {
  transitionKey: number;
  fallbackMs?: number;
};

// SceneTransitionOverlay renders a fixed full-viewport dark overlay that fades
// out over 700ms each time transitionKey increments from zero. The overlay is
// pointer-events: none so it never blocks interaction.
export function SceneTransitionOverlay({ transitionKey, fallbackMs = 2100 }: SceneTransitionOverlayProps) {
  const [visible, setVisible] = useState(false);
  const fallbackTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (transitionKey === 0) return;
    setVisible(true);

    // Fallback in case onAnimationEnd doesn't fire.
    fallbackTimerRef.current = setTimeout(() => {
      setVisible(false);
    }, fallbackMs);

    return () => {
      if (fallbackTimerRef.current) clearTimeout(fallbackTimerRef.current);
    };
  }, [transitionKey, fallbackMs]);

  const handleAnimationEnd = useCallback(() => {
    if (fallbackTimerRef.current) clearTimeout(fallbackTimerRef.current);
    setVisible(false);
  }, []);

  if (!visible) return null;

  return (
    <div
      key={transitionKey}
      className="scene-fade-overlay"
      onAnimationEnd={handleAnimationEnd}
      aria-hidden="true"
    />
  );
}
