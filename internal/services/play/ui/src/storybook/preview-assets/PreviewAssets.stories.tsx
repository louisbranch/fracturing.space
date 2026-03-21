import type { Meta, StoryObj } from "@storybook/react-vite";
import type { PreviewAvatarAsset, PreviewCampaignCoverAsset } from "./contract";
import {
  campaignCoverPreviewAssets,
  characterAvatarPreviewAssets,
  participantAvatarPreviewAssets,
} from "./fixtures";

function PreviewAssetsGallery() {
  return (
    <div className="preview-shell">
      <section className="space-y-2">
        <p className="preview-kicker">Storybook assets</p>
        <h1 className="font-display text-4xl text-base-content">Preview Assets</h1>
        <p className="preview-prose max-w-3xl">
          Stable participant avatars, character avatars, and campaign covers sourced from the
          checked-in asset catalogs for play-service mockups.
        </p>
      </section>

      <AssetSection
        assets={participantAvatarPreviewAssets}
        title="Participant avatars"
        description="Uses the published people-avatar catalog with the participant slot-1 crop."
        imageClassName="h-64 w-full object-cover"
      />

      <AssetSection
        assets={characterAvatarPreviewAssets}
        title="Character avatars"
        description="Uses the same published people-avatar catalog with deterministic character-slot crops."
        imageClassName="h-64 w-full object-cover"
      />

      <AssetSection
        assets={campaignCoverPreviewAssets}
        title="Campaign covers"
        description="Uses the full published campaign-cover catalog for future cover-bearing mockups."
        imageClassName="aspect-[3/2] w-full object-cover"
      />
    </div>
  );
}

function AssetSection(input: {
  assets: ReadonlyArray<PreviewAvatarAsset | PreviewCampaignCoverAsset>;
  title: string;
  description: string;
  imageClassName: string;
}) {
  return (
    <section className="space-y-4">
      <div className="space-y-1">
        <h2 className="font-display text-2xl text-base-content">{input.title}</h2>
        <p className="preview-prose">{input.description}</p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        {input.assets.map((asset) => (
          <article key={asset.id} className="preview-panel overflow-hidden">
            <figure className="bg-base-300/40">
              <img
                alt={`${asset.label} preview`}
                className={input.imageClassName}
                src={asset.imageUrl}
              />
            </figure>
            <div className="preview-panel-body gap-2">
              <h3 className="font-medium text-base-content">{asset.label}</h3>
              <p className="font-mono text-xs text-base-content/60">{asset.assetId}</p>
            </div>
          </article>
        ))}
      </div>
    </section>
  );
}

const meta = {
  title: "Reference/Preview Assets",
  component: PreviewAssetsGallery,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Catalog view for the shared Storybook preview assets used by play-service mockups.",
      },
    },
  },
} satisfies Meta<typeof PreviewAssetsGallery>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Catalog: Story = {};
