import type { PlayerConnectionState } from "../shared/contract";
import type { HUDHeaderProps } from "./contract";

type ConnectionPresentation = {
  label: string;
  statusClassName: string;
};

const connectionPresentation: Record<PlayerConnectionState, ConnectionPresentation> = {
  connecting: {
    label: "Connecting",
    statusClassName: "status status-info",
  },
  connected: {
    label: "Connected",
    statusClassName: "status status-success",
  },
  reconnecting: {
    label: "Reconnecting",
    statusClassName: "status status-warning",
  },
  disconnected: {
    label: "Disconnected",
    statusClassName: "status status-error",
  },
};

// HUDHeader is the thin player HUD bar that keeps campaign context, a web
// escape hatch, and connection state visible without occupying stage space.
export function HUDHeader({ campaignName, backURL, connection }: HUDHeaderProps) {
  const presentation = connectionPresentation[connection];

  return (
    <header className="preview-panel hud-header" aria-label="Player HUD header">
      <div className="grid min-h-11 grid-cols-[1fr_auto_1fr] items-center gap-2 px-2 py-1.5">
        <div className="flex min-w-0 justify-start">
          <a className="btn btn-neutral btn-sm" href={backURL}>
            Back To Campaign
          </a>
        </div>

        <div className="min-w-0 px-2 text-center">
          <p className="truncate font-display text-lg text-base-content sm:text-xl">{campaignName}</p>
        </div>

        <div className="flex shrink-0 justify-end">
          <div className="rounded-box border border-base-300/70 bg-base-100/70 px-2 py-1.5 text-sm text-base-content/75">
            <div className="flex items-center gap-2">
              <span className={presentation.statusClassName} />
              <span>{presentation.label}</span>
            </div>
          </div>
        </div>
      </div>
    </header>
  );
}
