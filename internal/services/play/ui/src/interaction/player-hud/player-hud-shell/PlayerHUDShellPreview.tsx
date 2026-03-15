import type { PlayerHUDState } from "../shared/contract";
import { usePlayerComposerPreviewState } from "../player-composer/usePlayerComposerPreviewState";
import { PlayerHUDShell } from "./PlayerHUDShell";

type PlayerHUDShellPreviewProps = {
  initialState: PlayerHUDState;
};

export function PlayerHUDShellPreview({ initialState }: PlayerHUDShellPreviewProps) {
  const { actions, state: composerState } = usePlayerComposerPreviewState(initialState.composer);

  return <PlayerHUDShell composerActions={actions} state={{ ...initialState, composer: composerState }} />;
}
