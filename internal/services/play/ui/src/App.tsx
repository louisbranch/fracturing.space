import type { AppMode } from "./app_mode";

const storybookURL = "http://localhost:6006";
const storybookCommand = "npm run storybook";
const storybookWorkspace = "internal/services/play/ui";

// App renders only runtime-oriented placeholder shells now that isolated
// component work has moved into the separate Storybook workflow.
export function App(props: { mode: AppMode }) {
  switch (props.mode.kind) {
    case "root-placeholder":
      return (
        <PlaceholderScreen
          kicker="Play UI shell"
          title="Play runtime UI deferred"
          body="The bundled play SPA no longer hosts isolated component previews. Run Storybook locally for Character Card design and review work."
        />
      );
    case "runtime-placeholder":
      return (
        <PlaceholderScreen
          kicker="Runtime placeholder"
          title="Play runtime UI deferred"
          body="Campaign routes remain reserved for the future runtime shell. Isolated component work now happens in the separate Storybook workflow."
          detailLabel="Campaign path"
          detailValue={`/campaigns/${props.mode.campaignId}`}
        />
      );
    case "unsupported":
      return (
        <main className="preview-shell">
          <section className="preview-panel max-w-3xl self-center">
            <div className="preview-panel-body items-start gap-4">
              <span className="preview-kicker">Unsupported route</span>
              <h1 className="font-display text-4xl">No UI mapped to this path</h1>
              <p className="preview-prose text-base">
                Use the separate Storybook workflow for component work or a campaign path for the runtime
                placeholder surface.
              </p>
              <div className="rounded-box border border-base-300/70 bg-base-100/85 px-4 py-3 text-sm text-base-content/80">
                Requested path: <code>{props.mode.path}</code>
              </div>
              <div className="flex flex-wrap gap-3">
                <a className="btn btn-primary" href={storybookURL} rel="noreferrer" target="_blank">
                  Open Storybook
                </a>
                <a className="btn btn-ghost" href="/campaigns/example-campaign">
                  View runtime placeholder
                </a>
              </div>
            </div>
          </section>
        </main>
      );
  }
}

function PlaceholderScreen(input: {
  kicker: string;
  title: string;
  body: string;
  detailLabel?: string;
  detailValue?: string;
}) {
  return (
    <main className="preview-shell">
      <section className="preview-panel max-w-3xl self-center">
        <div className="preview-panel-body items-start gap-4">
          <span className="preview-kicker">{input.kicker}</span>
          <h1 className="font-display text-4xl">{input.title}</h1>
          <p className="preview-prose text-base">{input.body}</p>
          {input.detailLabel && input.detailValue ? (
            <div className="rounded-box border border-base-300/70 bg-base-100/85 px-4 py-3 text-sm text-base-content/80">
              {input.detailLabel}: <code>{input.detailValue}</code>
            </div>
          ) : null}
          <div className="w-full rounded-box border border-base-300/70 bg-base-100/85 px-4 py-4 text-sm text-base-content/80">
            <p>
              Run <code>{storybookCommand}</code> from <code>{storybookWorkspace}</code>.
            </p>
            <p className="mt-2">
              Storybook URL: <code>{storybookURL}</code>
            </p>
          </div>
          <div className="flex flex-wrap gap-3">
            <a className="btn btn-primary" href={storybookURL} rel="noreferrer" target="_blank">
              Open Storybook
            </a>
            <a className="btn btn-ghost" href="/campaigns/example-campaign">
              View runtime placeholder
            </a>
          </div>
        </div>
      </section>
    </main>
  );
}
