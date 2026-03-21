import { RefreshCcw, Wifi, WifiOff } from "lucide-react";
import type { ReactNode } from "react";
import type { HUDConnectionState } from "../contract";
import type { HUDConnectionBadgeProps } from "./contract";

function connectionDisplay(connectionState: HUDConnectionState): {
  badgeClassName: string;
  icon: ReactNode;
  label: string;
  tooltip: string;
} {
  switch (connectionState) {
    case "connected":
      return {
        badgeClassName: "badge-success badge-soft",
        icon: <Wifi size={12} aria-hidden="true" />,
        label: "Connected",
        tooltip: "Realtime connection is live.",
      };
    case "reconnecting":
      return {
        badgeClassName: "badge-warning badge-soft",
        icon: <RefreshCcw size={12} aria-hidden="true" className="animate-spin" />,
        label: "Reconnecting",
        tooltip: "Attempting to restore realtime updates.",
      };
    case "disconnected":
      return {
        badgeClassName: "badge-error badge-soft",
        icon: <WifiOff size={12} aria-hidden="true" />,
        label: "Disconnected",
        tooltip: "Realtime connection is unavailable.",
      };
  }
}

export function HUDConnectionBadge({ connectionState }: HUDConnectionBadgeProps) {
  const display = connectionDisplay(connectionState);

  return (
    <span
      aria-label={`Connection status: ${display.label}`}
      className="tooltip tooltip-left shrink-0"
      data-tip={display.tooltip}
    >
      <span className={`badge badge-sm gap-1.5 ${display.badgeClassName}`} tabIndex={0}>
        {display.icon}
        <span aria-hidden="true" className="hidden sm:inline">
          {display.label}
        </span>
      </span>
    </span>
  );
}
