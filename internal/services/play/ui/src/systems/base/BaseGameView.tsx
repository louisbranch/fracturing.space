import type { SystemRendererProps } from "../../types";
import { aiTurnLabel, oocLabel, phaseLabel, sessionLabel } from "../../utils";

export function BaseGameView({ snapshot }: SystemRendererProps) {
  const state = snapshot.interaction_state;
  const scene = state.active_scene;
  const phase = state.player_phase;

  return (
    <section className="play-panel">
      <div className="play-panel-body">
        <div className="play-panel-head">
          <div className="space-y-2">
            <p className="play-eyebrow">Scene state</p>
            <h2 className="font-display text-4xl">{scene?.name || "No active scene"}</h2>
            <p className="play-prose">
              {scene?.description || "The GM has not opened a scene yet."}
            </p>
          </div>
          <div className="play-badge-row">
            <span className="badge badge-outline badge-info">{phaseLabel(phase?.status)}</span>
            <span className="badge badge-outline badge-warning">{oocLabel(state)}</span>
            <span className="badge badge-outline">AI: {aiTurnLabel(state)}</span>
          </div>
        </div>

        <div className="stats stats-vertical border border-base-300/70 bg-base-100 shadow sm:stats-horizontal">
          <div className="stat">
            <div className="stat-title">Session</div>
            <div className="stat-value text-lg">{sessionLabel(state.active_session)}</div>
          </div>
          <div className="stat">
            <div className="stat-title">Viewer</div>
            <div className="stat-value text-lg">{state.viewer?.name || "Unknown participant"}</div>
          </div>
          <div className="stat">
            <div className="stat-title">GM authority</div>
            <div className="stat-value text-lg">{state.gm_authority_participant_id || "Unassigned"}</div>
          </div>
        </div>

        <section className="space-y-3">
          <div className="flex items-center justify-between gap-3">
            <h3 className="font-display text-2xl">Characters in scene</h3>
            <span className="badge badge-ghost">{(scene?.characters ?? []).length} present</span>
          </div>
          <ul className="flex flex-wrap gap-2">
            {(scene?.characters ?? []).length > 0 ? (
              (scene?.characters ?? []).map((character) => (
                <li key={character.character_id} className="badge badge-lg badge-outline badge-accent px-4 py-4">
                  {character.name || character.character_id}
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
            <span className="badge badge-ghost">{(phase?.slots ?? []).length} tracked</span>
          </div>
          <ul className="space-y-3">
            {(phase?.slots ?? []).length > 0 ? (
              (phase?.slots ?? []).map((slot) => (
                <li
                  key={`${slot.participant_id}-${slot.updated_at || "slot"}`}
                  className="rounded-box border border-base-300/70 bg-base-100 px-4 py-3 shadow-sm"
                >
                  <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                    <strong>{slot.participant_id || "participant"}</strong>
                    <span className="badge badge-outline badge-sm">
                      {slot.updated_at ? "Updated" : "Pending"}
                    </span>
                  </div>
                  <p className="mt-2 text-sm leading-6 text-base-content/72">
                    {slot.summary_text || "Waiting for action."}
                  </p>
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
