import { useRef, useState } from "react";
import type { PlayerComposerState } from "../shared/contract";
import type { PlayerComposerActionHandlers } from "./contract";

type PlayerComposerPreviewState = {
  state: PlayerComposerState;
  actions: PlayerComposerActionHandlers;
  lastAction: string | null;
};

// usePlayerComposerPreviewState keeps Storybook and RTL previews interactive
// without leaking any runtime transport assumptions into the component slice.
export function usePlayerComposerPreviewState(initialState: PlayerComposerState): PlayerComposerPreviewState {
  const [state, setState] = useState<PlayerComposerState>(initialState);
  const [lastAction, setLastAction] = useState<string | null>(null);
  const previousSceneState = useRef(initialState.scene);

  const actions: PlayerComposerActionHandlers = {
    onModeChange: (mode) => {
      setState((currentState) => ({
        ...currentState,
        activeMode: mode,
      }));
    },
    onMinimizeChange: (minimized) => {
      setState((currentState) => ({
        ...currentState,
        minimized,
      }));
    },
    onDraftChange: (mode, draft) => {
      setState((currentState) => ({
        ...currentState,
        drafts: {
          ...currentState.drafts,
          [mode]: draft,
        },
      }));
    },
    onClearScratch: () => {
      setState((currentState) => ({
        ...currentState,
        drafts: {
          ...currentState.drafts,
          scratch: "",
        },
      }));
      setLastAction("Scratch pad cleared");
    },
    onSceneYieldToggle: () => {
      setState((currentState) => ({
        ...currentState,
        scene: {
          ...currentState.scene,
          yielded: !currentState.scene.yielded,
        },
      }));
    },
    onSceneSubmit: () => {
      setLastAction("Scene draft submitted");
    },
    onOOCPause: () => {
      setState((currentState) => {
        previousSceneState.current = currentState.scene;

        return {
          ...currentState,
          activeMode: "ooc",
          ooc: {
            open: true,
            helperText: "The table is paused. OOC messages can be posted until the scene resumes.",
          },
          scene: {
            ...currentState.scene,
            enabled: false,
            reason: "Scene play is paused while out-of-character discussion is open.",
          },
        };
      });
      setLastAction("Table paused");
    },
    onOOCResume: () => {
      setState((currentState) => ({
        ...currentState,
        ooc: {
          open: false,
          helperText: "Pause the table to open out-of-character discussion.",
        },
        scene: {
          ...previousSceneState.current,
        },
      }));
      setLastAction("Scene resumed");
    },
    onOOCSubmit: () => {
      setLastAction("OOC message submitted");
    },
    onChatSubmit: () => {
      setLastAction("Chat message submitted");
    },
  };

  return {
    state,
    actions,
    lastAction,
  };
}
