import type { BackstageStatusBannerProps } from "./contract";

function statusCopy(resumeState: BackstageStatusBannerProps["resumeState"], viewerReady: boolean): string {
  switch (resumeState) {
    case "waiting-on-gm":
      return "Waiting on GM.";
    case "collecting-ready":
      return viewerReady ? "You are ready." : "Awaiting player readiness.";
    default:
      return "OOC is closed.";
  }
}

export function BackstageStatusBanner({
  mode,
  resumeState,
  viewerReady,
  onViewerReadyToggle,
}: BackstageStatusBannerProps) {
  const open = mode === "open";
  const buttonLabel = viewerReady ? "Clear Ready" : "Mark Ready";

  return (
    <section
      aria-label="Backstage status"
      className="border-b border-base-300/70 bg-base-200/40 px-4 py-4"
    >
      <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
        <div className="flex flex-wrap items-center gap-2">
          <span className={`badge ${open ? "badge-warning badge-soft" : "badge-ghost"}`}>
            {open ? "OOC Open" : "Backstage Idle"}
          </span>
          <span className="text-sm font-medium text-base-content/75">
            {statusCopy(resumeState, viewerReady)}
          </span>
        </div>

        <button
          type="button"
          className={`btn ${viewerReady ? "btn-success" : "btn-outline"} self-start`}
          disabled={!open}
          onClick={onViewerReadyToggle}
        >
          {buttonLabel}
        </button>
      </div>
    </section>
  );
}
