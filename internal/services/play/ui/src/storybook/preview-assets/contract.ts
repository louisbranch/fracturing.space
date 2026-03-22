export type PreviewCatalogAsset = {
  id: string;
  label: string;
  setId: string;
  assetId: string;
  imageUrl: string;
};

export type PreviewAvatarCrop = {
  slot: number;
  x: number;
  y: number;
  widthPx: number;
  heightPx: number;
  deliveryWidthPx: number;
};

export type PreviewAvatarAsset = PreviewCatalogAsset & {
  crop: PreviewAvatarCrop;
};

export type PreviewPortraitSheetAsset = PreviewCatalogAsset & {
  widthPx: number;
  heightPx: number;
};

export type PreviewCampaignCoverAsset = PreviewCatalogAsset;
