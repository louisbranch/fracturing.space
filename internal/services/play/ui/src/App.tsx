import { ChatPanel } from "./features/chat/ChatPanel";
import { DraftPanel } from "./features/draft/DraftPanel";
import { usePlayRuntime } from "./runtime";
import { resolveSystemRenderer } from "./systems/registry";
import { createPlayShellViewModel, createSystemRenderViewModel } from "./view_models";

export function App() {
  const runtime = usePlayRuntime();
  const { state } = runtime;

  if (state.loading) {
    return (
      <main className="play-shell items-center justify-center">
        <section className="play-panel w-full max-w-2xl">
          <div className="play-panel-body items-center text-center">
            <span className="play-eyebrow">Connecting</span>
            <h1 className="font-display text-4xl">Opening the table...</h1>
            <p className="play-prose max-w-xl">
              Fetching the current scene, transcript, and live session state.
            </p>
            <span className="loading loading-bars loading-lg text-primary" aria-hidden="true" />
          </div>
        </section>
      </main>
    );
  }

  if (!state.bootstrap || !state.snapshot) {
    return (
      <main className="play-shell items-center justify-center">
        <section className="play-panel w-full max-w-2xl">
          <div className="play-panel-body items-center text-center">
            <span className="play-eyebrow">Unavailable</span>
            <h1 className="font-display text-4xl">Play surface unavailable</h1>
            <p className="play-prose max-w-xl">
              {state.error || "The play surface could not load the active campaign."}
            </p>
          </div>
        </section>
      </main>
    );
  }

  const renderer = resolveSystemRenderer(state.bootstrap.system);
  const shellView = createPlayShellViewModel(state.bootstrap, state.snapshot, state.connected);
  const systemView = createSystemRenderViewModel(state.snapshot);

  return (
    <main className="play-shell">
      <header className="play-hero">
        <div className="play-hero-body">
          <div className="max-w-3xl space-y-4">
            <span className="play-eyebrow">Active play</span>
            <h1 className="font-display text-4xl sm:text-5xl">{shellView.campaignName}</h1>
            <p className="play-prose text-base sm:text-lg">
              Live interaction state for {shellView.viewerName}.
            </p>
          </div>
          <div className="play-badge-row">
            <span className="badge badge-primary badge-outline badge-lg">{shellView.systemLabel}</span>
            <span className="badge badge-accent badge-outline badge-lg">{shellView.sessionLabel}</span>
            <span
              className={`badge badge-lg ${state.connected ? "badge-success" : "badge-error"} badge-outline`}
              data-testid="live-status"
            >
              {shellView.connectedLabel}
            </span>
          </div>
        </div>
      </header>

      {state.error ? (
        <div className="alert alert-error shadow-lg" role="alert">
          <span>{state.error}</span>
        </div>
      ) : null}

      <div className="play-workspace">
        <section className="play-column">
          {renderer.render({ bootstrap: state.bootstrap, snapshot: state.snapshot, view: systemView })}
        </section>
        <aside className="play-column">
          <div className="play-sticky">
            <ChatPanel
              connected={state.connected}
              loadingHistory={state.loadingHistory}
              messages={state.messages}
              typing={Object.values(state.chatTyping)}
              onLoadOlder={runtime.loadOlderMessages}
              onSend={runtime.sendChat}
              onTypingChange={runtime.setChatTyping}
            />
          </div>
          <DraftPanel
            typing={Object.values(state.draftTyping)}
            onTypingChange={runtime.setDraftTyping}
          />
        </aside>
      </div>
    </main>
  );
}
