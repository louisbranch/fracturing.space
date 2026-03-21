import { Maximize2, Minimize2 } from "lucide-react";
import { useId, type ReactNode } from "react";
import type { PlayerComposerMode } from "../shared/contract";
import type { PlayerComposerProps } from "./contract";

type ComposerModePresentation = {
  label: string;
};

const composerModePresentation: Record<PlayerComposerMode, ComposerModePresentation> = {
  scratch: {
    label: "Scratch Pad",
  },
  scene: {
    label: "Active Scene",
  },
  ooc: {
    label: "Out Of Character",
  },
  chat: {
    label: "Chat",
  },
};

const composerModes: PlayerComposerMode[] = ["scratch", "scene", "ooc", "chat"];

function submitDisabled(draft: string): boolean {
  return draft.trim().length === 0;
}

type ComposerSurfaceProps = {
  draft: string;
  mode: PlayerComposerMode;
  disabled?: boolean;
  helperText?: string;
  actions: ReactNode;
  onDraftChange?: (mode: PlayerComposerMode, draft: string) => void;
};

function ComposerSurface({ draft, mode, disabled = false, helperText, actions, onDraftChange }: ComposerSurfaceProps) {
  const labels: Record<PlayerComposerMode, string> = {
    scratch: "Scratch Pad draft",
    scene: "Active Scene draft",
    ooc: "Out Of Character draft",
    chat: "Chat draft",
  };

  return (
    <div className="hud-composer-surface">
      <div className="hud-composer-editor">
        {helperText ? <p className="text-sm leading-5 text-base-content/68">{helperText}</p> : null}
        <textarea
          aria-label={labels[mode]}
          className="textarea textarea-bordered hud-composer-textarea"
          disabled={disabled}
          onChange={(event) => onDraftChange?.(mode, event.target.value)}
          rows={4}
          style={{ width: "100%", maxWidth: "none", minWidth: 0 }}
          value={draft}
        />
      </div>
      <div className="hud-composer-actions">{actions}</div>
    </div>
  );
}

// PlayerComposer keeps the four drafting modes visible as one HUD surface while
// leaving state ownership to Storybook fixtures and future runtime adapters.
export function PlayerComposer({
  state,
  onModeChange,
  onMinimizeChange,
  onDraftChange,
  onClearScratch,
  onSceneYieldToggle,
  onSceneSubmit,
  onOOCPause,
  onOOCResume,
  onOOCSubmit,
  onChatSubmit,
}: PlayerComposerProps) {
  const baseId = useId();

  const activePanel = (() => {
    switch (state.activeMode) {
      case "scratch":
        return (
          <ComposerSurface
            actions={
              <button className="btn btn-neutral btn-sm" onClick={onClearScratch} type="button">
                Clear
              </button>
            }
            draft={state.drafts.scratch}
            mode="scratch"
            onDraftChange={onDraftChange}
          />
        );
      case "scene":
        return (
          <ComposerSurface
            actions={
              <>
                <button
                  className="btn btn-neutral btn-sm"
                  disabled={!state.scene.enabled}
                  onClick={onSceneYieldToggle}
                  type="button"
                >
                  {state.scene.yielded ? "Unyield" : "Yield"}
                </button>
                <button
                  className="btn btn-primary btn-sm"
                  disabled={!state.scene.enabled || submitDisabled(state.drafts.scene)}
                  onClick={onSceneSubmit}
                  type="button"
                >
                  Submit
                </button>
              </>
            }
            disabled={!state.scene.enabled}
            draft={state.drafts.scene}
            helperText={!state.scene.enabled ? state.scene.reason : undefined}
            mode="scene"
            onDraftChange={onDraftChange}
          />
        );
      case "ooc":
        return (
          <ComposerSurface
            actions={
              state.ooc.open ? (
                <>
                  <button className="btn btn-neutral btn-sm" onClick={onOOCResume} type="button">
                    Resume
                  </button>
                  <button
                    className="btn btn-primary btn-sm"
                    disabled={submitDisabled(state.drafts.ooc)}
                    onClick={onOOCSubmit}
                    type="button"
                  >
                    Submit
                  </button>
                </>
              ) : (
              <button
                className="btn btn-warning btn-sm"
                onClick={onOOCPause}
                type="button"
              >
                Pause
              </button>
              )
            }
            disabled={!state.ooc.open}
            draft={state.drafts.ooc}
            helperText={state.ooc.open ? state.ooc.helperText : undefined}
            mode="ooc"
            onDraftChange={onDraftChange}
          />
        );
      case "chat":
        return (
          <ComposerSurface
            actions={
              <button
                className="btn btn-primary btn-sm"
                disabled={submitDisabled(state.drafts.chat)}
                onClick={onChatSubmit}
                type="button"
              >
                Submit
              </button>
            }
            draft={state.drafts.chat}
            mode="chat"
            onDraftChange={onDraftChange}
          />
        );
    }
  })();

  return (
    <section aria-label="Player composer" className="preview-panel hud-composer">
      <div className="hud-composer-bar">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div aria-label="Player composer modes" className="tabs tabs-lift hud-composer-tabs" role="tablist">
            {composerModes.map((mode) => {
              const presentation = composerModePresentation[mode];
              const isActive = state.activeMode === mode;

              return (
                <button
                  aria-controls={`${baseId}-panel-${mode}`}
                  aria-selected={isActive}
                  className={`tab hud-composer-tab ${isActive ? "tab-active" : ""}`}
                  id={`${baseId}-tab-${mode}`}
                  key={mode}
                  onClick={() => onModeChange?.(mode)}
                  role="tab"
                  type="button"
                >
                  {presentation.label}
                </button>
              );
            })}
          </div>
          <div
            className="tooltip tooltip-left"
            data-tip={state.minimized ? "Maximize composer" : "Minimize composer"}
          >
            <button
              aria-label={state.minimized ? "Expand composer" : "Minimize composer"}
              className="hud-composer-toggle"
              onClick={() => onMinimizeChange?.(!state.minimized)}
              title={state.minimized ? "Expand composer" : "Minimize composer"}
              type="button"
            >
              {state.minimized ? <Maximize2 className="size-4" /> : <Minimize2 className="size-4" />}
            </button>
          </div>
        </div>
      </div>

      {state.minimized ? null : (
        <div className="hud-composer-body">
          <div
            aria-labelledby={`${baseId}-tab-${state.activeMode}`}
            className="hud-composer-panel"
            id={`${baseId}-panel-${state.activeMode}`}
            role="tabpanel"
          >
            {activePanel}
          </div>
        </div>
      )}
    </section>
  );
}
