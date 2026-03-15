import type { GMReviewPanelProps } from "./contract";

// GMReviewPanel isolates the GM-only review controls and summary copy so they
// can be previewed without the rest of the runtime shell.
export function GMReviewPanel({
  slots,
  onAcceptPhase,
  onRequestRevisions,
  onEndPhase,
}: GMReviewPanelProps) {
  const underReviewCount = slots.filter((slot) => slot.reviewStatus === "under_review").length;
  const revisionCount = slots.filter((slot) => slot.reviewStatus === "changes_requested").length;

  return (
    <section className="preview-panel" aria-label="GM review panel">
      <div className="preview-panel-body gap-4">
        <div>
          <span className="preview-kicker">GM Review</span>
          <h2 className="font-display text-2xl text-base-content">Resolve the Beat</h2>
          <p className="mt-2 text-sm leading-6 text-base-content/72">
            Review participant summaries, accept the beat, or return it with a narrower revision ask.
          </p>
        </div>

        <div className="grid gap-3 md:grid-cols-3">
          <ReviewStat label="Under Review" value={underReviewCount} />
          <ReviewStat label="Changes Requested" value={revisionCount} />
          <ReviewStat label="Total Slots" value={slots.length} />
        </div>

        <div className="flex flex-wrap gap-3">
          <button className="btn btn-primary" onClick={onAcceptPhase} type="button">
            Accept Phase
          </button>
          <button className="btn btn-secondary" onClick={onRequestRevisions} type="button">
            Request Revisions
          </button>
          <button className="btn btn-ghost" onClick={onEndPhase} type="button">
            End Phase
          </button>
        </div>
      </div>
    </section>
  );
}

function ReviewStat(input: { label: string; value: number }) {
  return (
    <div className="rounded-box border border-base-300/70 bg-base-100/65 px-4 py-3">
      <p className="text-xs uppercase tracking-[0.18em] text-base-content/45">{input.label}</p>
      <p className="mt-2 font-display text-3xl text-base-content">{input.value}</p>
    </div>
  );
}
