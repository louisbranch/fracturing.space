import type { PlayerComposerState } from "../shared/contract";
import { PlayerComposer } from "./PlayerComposer";
import { usePlayerComposerPreviewState } from "./usePlayerComposerPreviewState";

type PlayerComposerPreviewProps = {
  initialState: PlayerComposerState;
  showActionLog?: boolean;
};

export function PlayerComposerPreview({ initialState, showActionLog = false }: PlayerComposerPreviewProps) {
  const { actions, lastAction, state } = usePlayerComposerPreviewState(initialState);

  return (
    <div className="space-y-3">
      <PlayerComposer state={state} {...actions} />
      {showActionLog && lastAction ? (
        <p className="text-sm text-base-content/55" aria-live="polite">
          Preview action: {lastAction}
        </p>
      ) : null}
    </div>
  );
}
