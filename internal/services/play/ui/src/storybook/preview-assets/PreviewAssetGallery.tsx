import type { PreviewCatalogAsset } from "./contract";

type PreviewAssetGalleryProps = {
  pageTitle: string;
  pageDescription: string;
  sectionTitle: string;
  sectionDescription: string;
  assets: ReadonlyArray<PreviewCatalogAsset>;
  imageClassName: string;
};

export function PreviewAssetGallery(input: PreviewAssetGalleryProps) {
  return (
    <div className="preview-shell">
      <section className="space-y-2">
        <p className="preview-kicker">Storybook assets</p>
        <h1 className="font-display text-4xl text-base-content">{input.pageTitle}</h1>
        <p className="preview-prose max-w-3xl">{input.pageDescription}</p>
      </section>

      <section className="space-y-4">
        <div className="space-y-1">
          <h2 className="font-display text-2xl text-base-content">{input.sectionTitle}</h2>
          <p className="preview-prose">{input.sectionDescription}</p>
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
    </div>
  );
}
