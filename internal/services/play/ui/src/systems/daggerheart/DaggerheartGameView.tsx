import type { SystemRendererProps } from "../../types";
import { BaseGameView } from "../base/BaseGameView";

export function DaggerheartGameView(props: SystemRendererProps) {
  return (
    <div className="flex flex-col gap-6">
      <section className="play-panel overflow-hidden">
        <div className="play-panel-body gap-3 bg-gradient-to-r from-secondary/20 via-primary/10 to-base-200">
          <span className="play-eyebrow">System profile</span>
          <div className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
            <div>
              <h2 className="font-display text-3xl">Daggerheart live table</h2>
              <p className="play-prose">
                A system-specific surface can layer on top of the shared play shell without changing
                the transport contract.
              </p>
            </div>
            <span className="badge badge-secondary badge-outline badge-lg">Daggerheart</span>
          </div>
        </div>
      </section>
      <BaseGameView {...props} />
    </div>
  );
}
