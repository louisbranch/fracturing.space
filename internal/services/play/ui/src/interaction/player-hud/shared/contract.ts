// PlayerConnectionState captures the browser-side realtime state displayed in
// the HUD header.
export type PlayerConnectionState =
  | "connecting"
  | "connected"
  | "reconnecting"
  | "disconnected";

// PlayerComposerMode identifies the four drafting surfaces in the player HUD.
export type PlayerComposerMode = "scratch" | "scene" | "ooc" | "chat";

// PlayerComposerDrafts keeps each composer mode's text isolated so switching
// modes does not overwrite another draft.
export type PlayerComposerDrafts = Record<PlayerComposerMode, string>;

// PlayerStageState captures the stage viewport's placeholder and sample
// content without coupling to future runtime renderers.
export type PlayerStageState = {
  eyebrow?: string;
  title?: string;
  description?: string;
  content: string[];
  emptyMessage: string;
};

// PlayerSceneComposerState captures whether the player can submit to the active
// scene and whether they have already yielded.
export type PlayerSceneComposerState = {
  enabled: boolean;
  reason?: string;
  yielded: boolean;
};

// PlayerOOCComposerState captures the pause state and helper copy for the OOC
// composer mode.
export type PlayerOOCComposerState = {
  open: boolean;
  helperText?: string;
};

// PlayerComposerState keeps the HUD's local UI state explicit for Storybook and
// future runtime adapters.
export type PlayerComposerState = {
  activeMode: PlayerComposerMode;
  minimized: boolean;
  drafts: PlayerComposerDrafts;
  scene: PlayerSceneComposerState;
  ooc: PlayerOOCComposerState;
};

// PlayerHUDState is the composition-level fixture contract used by the player
// HUD shell and its child slices.
export type PlayerHUDState = {
  campaignName: string;
  backURL: string;
  connection: PlayerConnectionState;
  stage: PlayerStageState;
  composer: PlayerComposerState;
};
