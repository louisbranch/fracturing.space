import type { PlayerComposerActionHandlers } from "../player-composer/contract";
import type { PlayerHUDState } from "../shared/contract";

export type PlayerHUDShellProps = {
  state: PlayerHUDState;
  composerActions?: PlayerComposerActionHandlers;
};
