import type { PhaseStatusBannerProps } from "./contract";

type StatusConfig = {
  label: string;
  badgeClassName: string;
  description: string;
};

const statusConfig: Record<PhaseStatusBannerProps["phase"]["status"], StatusConfig> = {
  gm: {
    label: "GM",
    badgeClassName: "badge badge-warning badge-outline",
    description: "GM authority is live for the next authoritative table decision.",
  },
  players: {
    label: "Players",
    badgeClassName: "badge badge-info badge-outline",
    description: "The acting participants currently own the beat.",
  },
  gm_review: {
    label: "GM Review",
    badgeClassName: "badge badge-secondary badge-outline",
    description: "All acting participants have yielded and are waiting for review.",
  },
};

// PhaseStatusBanner keeps the workflow state visible without coupling any one
// surface to transport enums or shell-specific copy.
export function PhaseStatusBanner({ phase, viewerName, viewerRole }: PhaseStatusBannerProps) {
  const config = statusConfig[phase.status];

  return (
    <section className="rounded-box border border-base-300/70 bg-base-200/60 p-4" aria-label="Interaction phase status">
      <div className="flex flex-wrap items-center gap-2">
        <span className={config.badgeClassName}>{config.label}</span>
        {phase.oocOpen ? <span className="badge badge-accent badge-soft">OOC Open</span> : null}
        {viewerRole ? (
          <span className="badge badge-ghost badge-sm uppercase">{viewerRole}</span>
        ) : null}
      </div>
      <p className="mt-3 text-sm font-medium text-base-content/85">{phase.summary?.trim() || config.description}</p>
      <div className="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-sm text-base-content/60">
        <span>
          GM authority: <strong className="text-base-content/80">{phase.gmAuthorityName}</strong>
        </span>
        {viewerName ? (
          <span>
            Viewing as <strong className="text-base-content/80">{viewerName}</strong>
          </span>
        ) : null}
      </div>
    </section>
  );
}
