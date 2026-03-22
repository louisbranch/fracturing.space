import { createContext, useCallback, useContext, useMemo, useState } from "react";
import type { TransitionPreferences } from "./contract";
import { TRANSITION_PREFS_STORAGE_KEY, defaultTransitionPreferences } from "./contract";

type TransitionPreferencesContextValue = {
  preferences: TransitionPreferences;
  setPreference: <K extends keyof TransitionPreferences>(key: K, value: TransitionPreferences[K]) => void;
};

const TransitionPreferencesContext = createContext<TransitionPreferencesContextValue>({
  preferences: defaultTransitionPreferences,
  setPreference: () => {},
});

function loadPreferences(): TransitionPreferences {
  try {
    const raw = localStorage.getItem(TRANSITION_PREFS_STORAGE_KEY);
    if (!raw) return defaultTransitionPreferences;
    const parsed = JSON.parse(raw) as Partial<TransitionPreferences>;
    return {
      sceneVisual: typeof parsed.sceneVisual === "boolean" ? parsed.sceneVisual : defaultTransitionPreferences.sceneVisual,
      sceneSound: typeof parsed.sceneSound === "boolean" ? parsed.sceneSound : defaultTransitionPreferences.sceneSound,
      interactionVisual: typeof parsed.interactionVisual === "boolean" ? parsed.interactionVisual : defaultTransitionPreferences.interactionVisual,
      interactionSound: typeof parsed.interactionSound === "boolean" ? parsed.interactionSound : defaultTransitionPreferences.interactionSound,
    };
  } catch {
    return defaultTransitionPreferences;
  }
}

function persistPreferences(prefs: TransitionPreferences) {
  try {
    localStorage.setItem(TRANSITION_PREFS_STORAGE_KEY, JSON.stringify(prefs));
  } catch {
    // Storage full or unavailable — silently ignore.
  }
}

export function TransitionPreferencesProvider({ children }: { children: React.ReactNode }) {
  const [preferences, setPreferences] = useState<TransitionPreferences>(loadPreferences);

  const setPreference = useCallback(<K extends keyof TransitionPreferences>(key: K, value: TransitionPreferences[K]) => {
    setPreferences((current) => {
      const next = { ...current, [key]: value };
      persistPreferences(next);
      return next;
    });
  }, []);

  const contextValue = useMemo(
    () => ({ preferences, setPreference }),
    [preferences, setPreference],
  );

  return (
    <TransitionPreferencesContext value={contextValue}>
      {children}
    </TransitionPreferencesContext>
  );
}

export function useTransitionPreferences(): TransitionPreferencesContextValue {
  return useContext(TransitionPreferencesContext);
}
