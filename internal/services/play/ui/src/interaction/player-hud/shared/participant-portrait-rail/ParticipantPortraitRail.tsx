import type { ReactNode } from "react";
import { Check, Crown, RefreshCcw } from "lucide-react";
import type {
  ParticipantPortraitRailParticipant,
  ParticipantPortraitRailProps,
  ParticipantPortraitStatus,
} from "./contract";

function statusDisplay(status: ParticipantPortraitStatus): {
  badgeClassName: string;
  label: string;
  tooltip: string;
  icon: ReactNode;
} {
  switch (status) {
    case "typing":
      return {
        badgeClassName: "badge-info",
        label: "typing",
        tooltip: "Typing",
        icon: <span className="text-[10px] font-bold leading-none" aria-hidden="true">...</span>,
      };
    case "ready":
      return {
        badgeClassName: "badge-success",
        label: "ready",
        tooltip: "Ready to resume",
        icon: <Check size={12} aria-hidden="true" />,
      };
    case "active":
      return {
        badgeClassName: "badge-primary",
        label: "active",
        tooltip: "Acting now",
        icon: <span className="block h-2.5 w-2.5 rounded-full bg-base-100" aria-hidden="true" />,
      };
    case "yielded":
      return {
        badgeClassName: "badge-secondary",
        label: "yielded",
        tooltip: "Yielded",
        icon: <Check size={12} aria-hidden="true" />,
      };
    case "changes-requested":
      return {
        badgeClassName: "badge-warning",
        label: "changes requested",
        tooltip: "Changes requested",
        icon: <RefreshCcw size={12} aria-hidden="true" />,
      };
    default:
      return {
        badgeClassName: "badge-ghost",
        label: "waiting",
        tooltip: "Waiting",
        icon: <span className="block h-2 w-2 rounded-full bg-base-content/40" aria-hidden="true" />,
      };
  }
}

function roleLabel(participant: ParticipantPortraitRailParticipant): string | null {
  const value = participant.roleLabel?.trim();
  return value ? value : null;
}

export function ParticipantPortraitRail({
  participants,
  viewerParticipantId,
  ariaLabel = "Participant portraits",
}: ParticipantPortraitRailProps) {
  return (
    <aside
      aria-label={ariaLabel}
      className="flex w-24 shrink-0 flex-col items-center gap-3 border-l border-base-300/70 bg-base-200/25 px-2 py-3"
    >
      {participants.map((participant) => {
        const status = statusDisplay(participant.status);
        const isViewer = participant.id === viewerParticipantId;
        const fallback = participant.name.charAt(0).toUpperCase();
        const secondaryLabel = roleLabel(participant);

        return (
          <div
            key={participant.id}
            aria-label={`${participant.name}: ${status.label}`}
            className="flex flex-col items-center gap-1.5"
          >
            <div
              className={`relative aspect-[2/3] w-14 ${
                isViewer ? "ring-2 ring-primary ring-offset-2 ring-offset-base-100" : ""
              }`}
            >
              <div className="absolute inset-0 overflow-hidden border border-base-300 bg-base-300 text-base-content shadow-sm">
                {participant.avatarUrl ? (
                  <img
                    src={participant.avatarUrl}
                    alt={participant.name}
                    className="h-full w-full object-cover"
                  />
                ) : (
                  <div className="flex h-full items-center justify-center font-semibold">
                    {fallback}
                  </div>
                )}
              </div>
              {participant.ownsGMAuthority ? (
                <span
                  aria-label={`${participant.name} GM authority`}
                  className="tooltip tooltip-left absolute top-1 right-1 z-10"
                  data-tip="Owns GM authority"
                >
                  <span className="badge badge-warning badge-xs border border-base-100">
                    <Crown size={12} aria-hidden="true" />
                  </span>
                </span>
              ) : null}
              <span
                aria-label={`${participant.name} status: ${status.tooltip}`}
                className="tooltip tooltip-left absolute right-1 bottom-1 z-10"
                data-tip={status.tooltip}
              >
                <span
                  className={`badge badge-xs border border-base-100 ${status.badgeClassName}`}
                >
                  {status.icon}
                </span>
              </span>
            </div>
            <div className="text-center text-[10px] leading-tight text-base-content/70">
              <div className="font-medium text-base-content">{participant.name}</div>
              {secondaryLabel ? (
                <div className="uppercase tracking-wide">{secondaryLabel}</div>
              ) : null}
            </div>
          </div>
        );
      })}
    </aside>
  );
}
