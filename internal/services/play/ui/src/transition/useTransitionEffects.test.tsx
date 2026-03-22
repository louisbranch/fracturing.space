import { describe, expect, it } from "vitest";
import { renderHook } from "@testing-library/react";
import { TransitionPreferencesProvider } from "./TransitionPreferencesContext";
import { useTransitionEffects } from "./useTransitionEffects";

function wrapper({ children }: { children: React.ReactNode }) {
  return <TransitionPreferencesProvider>{children}</TransitionPreferencesProvider>;
}

describe("useTransitionEffects", () => {
  it("skips effects on initial mount", () => {
    const { result } = renderHook(
      () => useTransitionEffects("scene-1", "interaction-1"),
      { wrapper },
    );

    expect(result.current.sceneTransitionKey).toBe(0);
    expect(result.current.interactionTransitionActive).toBe(false);
  });

  it("fires scene transition when scene ID changes", () => {
    const { result, rerender } = renderHook(
      ({ sceneId, interactionId }) => useTransitionEffects(sceneId, interactionId),
      { wrapper, initialProps: { sceneId: "scene-1", interactionId: "interaction-1" } },
    );

    rerender({ sceneId: "scene-2", interactionId: "interaction-2" });

    expect(result.current.sceneTransitionKey).toBe(1);
    // Invariant: scene change is the superset, so interaction transition should not fire.
    expect(result.current.interactionTransitionActive).toBe(false);
  });

  it("fires interaction transition when only interaction ID changes", () => {
    const { result, rerender } = renderHook(
      ({ sceneId, interactionId }) => useTransitionEffects(sceneId, interactionId),
      { wrapper, initialProps: { sceneId: "scene-1", interactionId: "interaction-1" } },
    );

    rerender({ sceneId: "scene-1", interactionId: "interaction-2" });

    expect(result.current.sceneTransitionKey).toBe(0);
    expect(result.current.interactionTransitionActive).toBe(true);
  });

  it("increments scene key on successive scene changes", () => {
    const { result, rerender } = renderHook(
      ({ sceneId, interactionId }) => useTransitionEffects(sceneId, interactionId),
      { wrapper, initialProps: { sceneId: "scene-1", interactionId: "interaction-1" } },
    );

    rerender({ sceneId: "scene-2", interactionId: "interaction-1" });
    expect(result.current.sceneTransitionKey).toBe(1);

    rerender({ sceneId: "scene-3", interactionId: "interaction-1" });
    expect(result.current.sceneTransitionKey).toBe(2);
  });

  it("does not fire when IDs remain the same", () => {
    const { result, rerender } = renderHook(
      ({ sceneId, interactionId }) => useTransitionEffects(sceneId, interactionId),
      { wrapper, initialProps: { sceneId: "scene-1", interactionId: "interaction-1" } },
    );

    rerender({ sceneId: "scene-1", interactionId: "interaction-1" });

    expect(result.current.sceneTransitionKey).toBe(0);
    expect(result.current.interactionTransitionActive).toBe(false);
  });
});
