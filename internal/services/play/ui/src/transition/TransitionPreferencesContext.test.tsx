import { describe, expect, it } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { TransitionPreferencesProvider, useTransitionPreferences } from "./TransitionPreferencesContext";
import { defaultTransitionPreferences } from "./contract";

function wrapper({ children }: { children: React.ReactNode }) {
  return <TransitionPreferencesProvider>{children}</TransitionPreferencesProvider>;
}

describe("TransitionPreferencesContext", () => {
  it("returns default preferences on mount", () => {
    const { result } = renderHook(() => useTransitionPreferences(), { wrapper });
    expect(result.current.preferences).toEqual(defaultTransitionPreferences);
  });

  it("updates a single preference via setPreference", () => {
    const { result } = renderHook(() => useTransitionPreferences(), { wrapper });

    act(() => {
      result.current.setPreference("sceneSound", false);
    });

    expect(result.current.preferences.sceneSound).toBe(false);
    expect(result.current.preferences.sceneVisual).toBe(true);
  });

  it("preserves other keys when toggling one preference", () => {
    const { result } = renderHook(() => useTransitionPreferences(), { wrapper });

    act(() => {
      result.current.setPreference("interactionVisual", false);
    });
    act(() => {
      result.current.setPreference("interactionSound", false);
    });

    expect(result.current.preferences).toEqual({
      sceneVisual: true,
      sceneSound: true,
      interactionVisual: false,
      interactionSound: false,
    });
  });
});
