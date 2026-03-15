import type { SystemRendererProps } from "../../types";

export function BaseGameView({ view }: SystemRendererProps) {
  const scene = view.scene;
  const slots = view.slots;

  return (
    <section className="play-panel">
      <div className="play-panel-body">
        <div className="play-panel-head">
          <div className="space-y-2">
            <p className="play-eyebrow">Scene state</p>
            <h2 className="font-display text-4xl">{scene.title}</h2>
            <p className="play-prose">{scene.description}</p>
          </div>
          <div className="play-badge-row">
            <span className="badge badge-outline badge-info">{view.scenePhaseLabel}</span>
            <span className="badge badge-outline badge-warning">{view.oocLabel}</span>
            <span className="badge badge-outline">AI: {view.aiTurnLabel}</span>
          </div>
        </div>

        <div className="stats stats-vertical border border-base-300/70 bg-base-100 shadow sm:stats-horizontal">
          <div className="stat">
            <div className="stat-title">Session</div>
            <div className="stat-value text-lg">{view.sessionLabel}</div>
          </div>
          <div className="stat">
            <div className="stat-title">Viewer</div>
            <div className="stat-value text-lg">{view.viewerName}</div>
          </div>
          <div className="stat">
            <div className="stat-title">GM authority</div>
            <div className="stat-value text-lg">{view.gmAuthorityLabel}</div>
          </div>
        </div>

        <section className="space-y-3">
          <div className="flex items-center justify-between gap-3">
            <h3 className="font-display text-2xl">Characters in scene</h3>
            <span className="badge badge-ghost">{scene.characterCount} present</span>
          </div>
          <ul className="flex flex-wrap gap-2">
            {scene.characters.length > 0 ? (
              scene.characters.map((character) => (
                <li key={character.id} className="badge badge-lg badge-outline badge-accent px-4 py-4">
                  {character.name}
                </li>
              ))
            ) : (
              <li className="text-sm text-base-content/60">No characters are attached to the current scene.</li>
            )}
          </ul>
        </section>

        <section className="space-y-3">
          <div className="flex items-center justify-between gap-3">
            <h3 className="font-display text-2xl">Player slots</h3>
            <span className="badge badge-ghost">{slots.length} tracked</span>
          </div>
          <ul className="space-y-3">
            {slots.length > 0 ? (
              slots.map((slot) => (
                <li
                  key={slot.key}
                  className="rounded-box border border-base-300/70 bg-base-100 px-4 py-3 shadow-sm"
                >
                  <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                    <strong>{slot.participantLabel}</strong>
                    <span className="badge badge-outline badge-sm">{slot.statusLabel}</span>
                  </div>
                  <p className="mt-2 text-sm leading-6 text-base-content/72">{slot.summaryText}</p>
                </li>
              ))
            ) : (
              <li className="rounded-box border border-dashed border-base-300 bg-base-100/40 px-4 py-5 text-sm text-base-content/60">
                No player slots have been opened yet.
              </li>
            )}
          </ul>
        </section>
      </div>
    </section>
  );
}
