import type { PlayerHUDStatusBadge } from "./view-models";

type PlayerHUDStatusPillProps = {
  ariaLabel: string;
  status: PlayerHUDStatusBadge;
};

export function PlayerHUDStatusPill({ ariaLabel, status }: PlayerHUDStatusPillProps) {
  return (
    <span
      aria-label={ariaLabel}
      className="tooltip tooltip-left shrink-0"
      data-tip={status.tooltip}
    >
      <span className={`badge ${status.className} gap-1.5`} tabIndex={0}>
        {status.indicator === "loading-bars" ? (
          <span
            aria-hidden="true"
            className="loading loading-bars loading-xs shrink-0"
          />
        ) : null}
        <span>{status.label}</span>
      </span>
    </span>
  );
}
