import type { OOCOverlayPanelProps } from "./contract";

// OOCOverlayPanel captures the explicit out-of-character pause mode rather than
// treating it as normal chat with different copy.
export function OOCOverlayPanel({ phase, ooc, onResume }: OOCOverlayPanelProps) {
  return (
    <section className="preview-panel" aria-label="OOC overlay">
      <div className="preview-panel-body gap-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <span className="preview-kicker">Out Of Character</span>
            <h2 className="font-display text-2xl text-base-content">Table Pause</h2>
          </div>
          <span className="badge badge-accent badge-soft">{phase.gmAuthorityName} can resume</span>
        </div>

        <div className="rounded-box border border-accent/30 bg-accent/10 px-4 py-3">
          <p className="text-xs uppercase tracking-[0.18em] text-accent/80">Pause Reason</p>
          <p className="mt-2 text-sm leading-6 text-base-content/85">{ooc.reason}</p>
        </div>

        <div className="space-y-3">
          {ooc.posts.map((post) => (
            <article key={post.postId} className="rounded-box border border-base-300/70 bg-base-100/65 px-4 py-3">
              <div className="flex flex-wrap items-center gap-2">
                <strong className="text-sm text-base-content">{post.participantName}</strong>
                {post.emphasis === "gm" ? <span className="badge badge-warning badge-xs">GM</span> : null}
              </div>
              <p className="mt-2 text-sm leading-6 text-base-content/80">{post.body}</p>
            </article>
          ))}
        </div>

        <div className="rounded-box border border-base-300/70 bg-base-100/65 px-4 py-3">
          <p className="text-xs uppercase tracking-[0.18em] text-base-content/45">Ready To Resume</p>
          <div className="mt-3 flex flex-wrap gap-2">
            {ooc.readyParticipantNames.map((name) => (
              <span key={name} className="badge badge-success badge-soft">
                {name}
              </span>
            ))}
          </div>
        </div>

        <div className="flex flex-wrap gap-3">
          <button className="btn btn-primary" onClick={onResume} type="button">
            Resume Scene
          </button>
        </div>
      </div>
    </section>
  );
}
