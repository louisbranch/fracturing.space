import { HUDHeader } from "../hud-header/HUDHeader";
import { PlayerComposer } from "../player-composer/PlayerComposer";
import { StageViewport } from "../stage-viewport/StageViewport";
import type { PlayerHUDShellProps } from "./contract";

// PlayerHUDShell is the composition-only viewport that proves the fixed HUD
// layout before any live player interface or transport adapter exists.
export function PlayerHUDShell({ state, composerActions }: PlayerHUDShellProps) {
  return (
    <main aria-label="Player HUD shell" className="hud-shell">
      <HUDHeader backURL={state.backURL} campaignName={state.campaignName} connection={state.connection} />

      <div className="hud-shell-main">
        <StageViewport stage={state.stage} />
        <PlayerComposer state={state.composer} {...composerActions} />
      </div>
    </main>
  );
}
