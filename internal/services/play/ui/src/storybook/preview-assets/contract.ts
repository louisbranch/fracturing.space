export type PreviewAvatarCrop = {
  slot: number;
  x: number;
  y: number;
  widthPx: number;
  heightPx: number;
  deliveryWidthPx: number;
};

export type PreviewAvatarAsset = {
  id: string;
  label: string;
  setId: string;
  assetId: string;
  imageUrl: string;
  crop: PreviewAvatarCrop;
};

export type PreviewCampaignCoverAsset = {
  id: string;
  label: string;
  setId: string;
  assetId: string;
  imageUrl: string;
};
