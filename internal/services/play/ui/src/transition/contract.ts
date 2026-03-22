export type TransitionPreferences = {
  sceneVisual: boolean;
  sceneSound: boolean;
  interactionVisual: boolean;
  interactionSound: boolean;
};

export const defaultTransitionPreferences: TransitionPreferences = {
  sceneVisual: true,
  sceneSound: true,
  interactionVisual: true,
  interactionSound: true,
};

export const TRANSITION_PREFS_STORAGE_KEY = "fs:play:transition-prefs";
