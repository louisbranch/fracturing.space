import type { ActingSetRailProps } from "./contract";

// ActingSetRail makes the current acting set visible independently from slot
// progress so shell and scene stories can reuse it.
export function ActingSetRail({ actingSet }: ActingSetRailProps) {
  return (
    <section className="preview-panel" aria-label="Acting set">
      <div className="preview-panel-body gap-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div>
            <span className="preview-kicker">Acting Set</span>
            <h2 className="font-display text-2xl text-base-content">Who Owns This Beat</h2>
          </div>
          <span className="badge badge-outline">{actingSet.length} acting</span>
        </div>

        {actingSet.length === 0 ? (
          <p className="text-sm text-base-content/60">No acting characters are assigned to the current beat.</p>
        ) : (
          <div className="grid gap-3 md:grid-cols-2">
            {actingSet.map((entry) => (
              <article
                key={entry.id}
                className={`rounded-box border px-4 py-3 ${
                  entry.spotlight
                    ? "border-warning/40 bg-warning/10"
                    : "border-base-300/70 bg-base-100/65"
                }`}
              >
                <div className="flex items-center justify-between gap-3">
                  <h3 className="font-display text-xl text-base-content">{entry.name}</h3>
                  {entry.spotlight ? <span className="badge badge-warning badge-soft">Spotlight</span> : null}
                </div>
                <p className="mt-2 text-sm text-base-content/65">Controlled by {entry.participantName}</p>
              </article>
            ))}
          </div>
        )}
      </div>
    </section>
  );
}
