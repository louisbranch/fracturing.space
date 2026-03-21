import type { AppMode } from "./app_mode";
import { PlayRuntime } from "./PlayRuntime";
import type { PlayShellConfig } from "./shell_config";

const storybookURL = "http://localhost:6006";
const storybookCommand = "npm run storybook";
const storybookWorkspace = "internal/services/play/ui";

// App renders the runtime play shell when a campaign is loaded, or placeholder
// screens for other routes.
export function App(props: { mode: AppMode; shellConfig?: PlayShellConfig | null }) {
  switch (props.mode.kind) {
    case "root-placeholder":
      return (
        <PlaceholderScreen
          kicker="Play UI shell"
          title="Play runtime UI deferred"
          body="The bundled play SPA no longer hosts isolated component previews. Run Storybook locally for interaction workflow slices and Daggerheart reference components."
          shellConfig={props.shellConfig ?? null}
        />
      );
    case "runtime": {
      const config = props.shellConfig && props.shellConfig.campaignId === props.mode.campaignId
        ? props.shellConfig
        : null;
      if (!config) {
        return (
          <PlaceholderScreen
            kicker="Configuration missing"
            title="Play shell config not available"
            body="The play session shell config was not embedded by the server. This usually means the session cookie or launch grant is missing."
            detailLabel="Campaign path"
            detailValue={`/campaigns/${props.mode.campaignId}`}
            shellConfig={null}
          />
        );
      }
      return <PlayRuntime shellConfig={config} />;
    }
    case "runtime-placeholder":
      return (
        <PlaceholderScreen
          kicker="Runtime placeholder"
          title="Play runtime UI deferred"
          body="Campaign routes remain reserved for the future runtime shell. Isolated interaction and character component work now happens in the separate Storybook workflow."
          detailLabel="Campaign path"
          detailValue={`/campaigns/${props.mode.campaignId}`}
          shellConfig={
            props.shellConfig && props.shellConfig.campaignId === props.mode.campaignId ? props.shellConfig : null
          }
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
                Use the separate Storybook workflow for interaction and character component work or a
                campaign path for the runtime placeholder surface.
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
  shellConfig?: PlayShellConfig | null;
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
          {input.shellConfig ? (
            <div className="w-full rounded-box border border-base-300/70 bg-base-100/85 px-4 py-4 text-sm text-base-content/80">
              <p>
                Bootstrap endpoint: <code>{input.shellConfig.bootstrapPath || "(not configured)"}</code>
              </p>
              <p className="mt-2">
                Realtime endpoint: <code>{input.shellConfig.realtimePath || "(not configured)"}</code>
              </p>
              <p className="mt-2">
                Back link: <code>{input.shellConfig.backURL || "(not configured)"}</code>
              </p>
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
