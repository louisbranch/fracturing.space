import type { PlayPlayerSlotReviewStatus } from "../shared/contract";
import type { PlayerSlotBoardProps } from "./contract";

const reviewStatusCopy: Record<PlayPlayerSlotReviewStatus, { badge: string; className: string }> = {
  open: { badge: "Open", className: "badge badge-ghost badge-sm" },
  under_review: { badge: "Under Review", className: "badge badge-secondary badge-sm" },
  accepted: { badge: "Accepted", className: "badge badge-success badge-sm" },
  changes_requested: { badge: "Changes Requested", className: "badge badge-error badge-sm" },
};

// PlayerSlotBoard keeps participant-owned beat progress visible without mixing
// it with GM action controls.
export function PlayerSlotBoard({ slots }: PlayerSlotBoardProps) {
  return (
    <section className="preview-panel" aria-label="Player slot board">
      <div className="preview-panel-body gap-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div>
            <span className="preview-kicker">Player Slots</span>
            <h2 className="font-display text-2xl text-base-content">Committed Beat Progress</h2>
          </div>
          <span className="badge badge-outline">{slots.length} slots</span>
        </div>

        {slots.length === 0 ? (
          <p className="text-sm text-base-content/60">No acting participant slots exist for the current state.</p>
        ) : (
          <div className="grid gap-4 xl:grid-cols-2">
            {slots.map((slot) => {
              const reviewConfig = slot.reviewStatus ? reviewStatusCopy[slot.reviewStatus] : null;
              return (
                <article key={slot.participantId} className="rounded-box border border-base-300/70 bg-base-100/65 p-4">
                  <div className="flex flex-wrap items-center gap-2">
                    <h3 className="font-display text-xl text-base-content">{slot.participantName}</h3>
                    {slot.isViewer ? <span className="badge badge-info badge-soft">Viewer</span> : null}
                    {slot.yielded ? <span className="badge badge-warning badge-soft">Yielded</span> : null}
                    {reviewConfig ? <span className={reviewConfig.className}>{reviewConfig.badge}</span> : null}
                  </div>

                  <div className="mt-3 flex flex-wrap gap-2">
                    {slot.characterNames.map((name) => (
                      <span key={`${slot.participantId}-${name}`} className="badge badge-outline badge-sm">
                        {name}
                      </span>
                    ))}
                  </div>

                  <p className="mt-4 text-sm leading-7 text-base-content/85">
                    {slot.summaryText?.trim() || "No committed summary yet."}
                  </p>

                  {slot.reviewReason ? (
                    <div className="mt-4 rounded-box border border-error/30 bg-error/10 px-3 py-3 text-sm text-base-content/80">
                      <p className="text-xs uppercase tracking-[0.18em] text-error/80">Review Note</p>
                      <p className="mt-2">{slot.reviewReason}</p>
                    </div>
                  ) : null}
                </article>
              );
            })}
          </div>
        )}
      </div>
    </section>
  );
}
