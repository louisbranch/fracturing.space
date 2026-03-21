import avatarCatalogJSON from "../../../../../../platform/assets/catalog/data/avatars.v1.json";
import campaignCoverCatalogJSON from "../../../../../../platform/assets/catalog/data/campaign_covers.v1.json";
import cloudinaryCatalogJSON from "../../../../../../platform/assets/catalog/data/cloudinary_assets.high_fantasy.v1.json";
import type { PreviewAvatarAsset, PreviewAvatarCrop, PreviewCampaignCoverAsset } from "./contract";

type AssetSetJSON = {
  id: string;
  asset_ids: string[];
};

type AvatarPortraitJSON = {
  slot: number;
  x: number;
  y: number;
  width_px: number;
  height_px: number;
};

type AvatarSheetJSON = {
  set_id: string;
  portraits: AvatarPortraitJSON[];
};

type AvatarCatalogJSON = {
  manifest: {
    sets: AssetSetJSON[];
  };
  sheets: AvatarSheetJSON[];
};

type CampaignCoverCatalogJSON = {
  sets: AssetSetJSON[];
};

const PEOPLE_AVATAR_SET_ID = "avatar_set_v1";
const CAMPAIGN_COVER_SET_ID = "campaign_cover_set_v1";
const PARTICIPANT_SLOT = 1;
const CHARACTER_SLOT_CANDIDATES = [2, 3, 4] as const;
const AVATAR_DELIVERY_WIDTH_PX = 384;

const avatarCatalog = avatarCatalogJSON as AvatarCatalogJSON;
const campaignCoverCatalog = campaignCoverCatalogJSON as CampaignCoverCatalogJSON;
const cloudinarySecureURLs = decodeCloudinarySecureURLs(cloudinaryCatalogJSON as Record<string, unknown>);
const avatarSheet = requiredAvatarSheet(PEOPLE_AVATAR_SET_ID);

export const participantAvatarPreviewAssets: ReadonlyArray<PreviewAvatarAsset> =
  buildParticipantAvatarPreviewAssets();

export const characterAvatarPreviewAssets: ReadonlyArray<PreviewAvatarAsset> =
  buildCharacterAvatarPreviewAssets();

export const campaignCoverPreviewAssets: ReadonlyArray<PreviewCampaignCoverAsset> =
  buildCampaignCoverPreviewAssets();

function buildParticipantAvatarPreviewAssets(): ReadonlyArray<PreviewAvatarAsset> {
  return orderedAssetIDs(avatarCatalog.manifest.sets, PEOPLE_AVATAR_SET_ID).map((assetId) =>
    buildAvatarPreviewAsset(assetId, portraitCropForSlot(PARTICIPANT_SLOT)),
  );
}

function buildCharacterAvatarPreviewAssets(): ReadonlyArray<PreviewAvatarAsset> {
  return orderedAssetIDs(avatarCatalog.manifest.sets, PEOPLE_AVATAR_SET_ID).map((assetId, index) => {
    const entityId = `preview-character-${String(index + 1).padStart(2, "0")}`;
    const slot = deterministicAvatarSlot("character", entityId, CHARACTER_SLOT_CANDIDATES);
    return buildAvatarPreviewAsset(assetId, portraitCropForSlot(slot));
  });
}

function buildCampaignCoverPreviewAssets(): ReadonlyArray<PreviewCampaignCoverAsset> {
  return orderedAssetIDs(campaignCoverCatalog.sets, CAMPAIGN_COVER_SET_ID).map((assetId) => ({
    id: `${CAMPAIGN_COVER_SET_ID}:${assetId}`,
    label: humanizeAssetID(assetId),
    setId: CAMPAIGN_COVER_SET_ID,
    assetId,
    imageUrl: requiredCloudinaryURL(CAMPAIGN_COVER_SET_ID, assetId),
  }));
}

function buildAvatarPreviewAsset(assetId: string, crop: PreviewAvatarCrop): PreviewAvatarAsset {
  return {
    id: `${PEOPLE_AVATAR_SET_ID}:${assetId}`,
    label: humanizeAssetID(assetId),
    setId: PEOPLE_AVATAR_SET_ID,
    assetId,
    imageUrl: cropCloudinaryImageURL(requiredCloudinaryURL(PEOPLE_AVATAR_SET_ID, assetId), crop),
    crop,
  };
}

function orderedAssetIDs(sets: AssetSetJSON[], setId: string): string[] {
  const assetSet = sets.find((entry) => entry.id === setId);
  if (!assetSet) {
    throw new Error(`Missing asset set ${setId}`);
  }
  return [...assetSet.asset_ids];
}

function requiredAvatarSheet(setId: string): AvatarSheetJSON {
  const sheet = avatarCatalog.sheets.find((entry) => entry.set_id === setId);
  if (!sheet) {
    throw new Error(`Missing avatar sheet for ${setId}`);
  }
  return sheet;
}

function portraitCropForSlot(slot: number): PreviewAvatarCrop {
  const portrait = avatarSheet.portraits.find((entry) => entry.slot === slot);
  if (!portrait) {
    throw new Error(`Missing avatar portrait slot ${slot}`);
  }
  return {
    slot,
    x: portrait.x,
    y: portrait.y,
    widthPx: portrait.width_px,
    heightPx: portrait.height_px,
    deliveryWidthPx: AVATAR_DELIVERY_WIDTH_PX,
  };
}

function requiredCloudinaryURL(setId: string, assetId: string): string {
  const lookupKey = cloudinaryLookupKey(setId, assetId);
  const imageURL = cloudinarySecureURLs.get(lookupKey);
  if (!imageURL) {
    throw new Error(`Missing cloudinary URL for ${setId}:${assetId}`);
  }
  return imageURL;
}

function cloudinaryLookupKey(setId: string, assetId: string): string {
  return `${setId}\u0000${assetId}`;
}

function decodeCloudinarySecureURLs(raw: Record<string, unknown>): Map<string, string> {
  const out = new Map<string, string>();

  for (const value of Object.values(raw)) {
    if (!Array.isArray(value)) {
      continue;
    }
    for (const entry of value) {
      if (!isCloudinaryAssetEntry(entry)) {
        continue;
      }
      out.set(cloudinaryLookupKey(entry.set_id, entry.fs_asset_id), entry.cloudinary.secure_url);
    }
  }

  return out;
}

function isCloudinaryAssetEntry(
  value: unknown,
): value is { set_id: string; fs_asset_id: string; cloudinary: { secure_url: string } } {
  if (!value || typeof value !== "object") {
    return false;
  }

  const entry = value as {
    set_id?: unknown;
    fs_asset_id?: unknown;
    cloudinary?: { secure_url?: unknown };
  };

  return (
    typeof entry.set_id === "string" &&
    typeof entry.fs_asset_id === "string" &&
    typeof entry.cloudinary?.secure_url === "string"
  );
}

function cropCloudinaryImageURL(imageURL: string, crop: PreviewAvatarCrop): string {
  const transform = [
    `c_crop,w_${crop.widthPx},h_${crop.heightPx},x_${crop.x},y_${crop.y}`,
    `f_auto,q_auto,dpr_auto,c_limit,w_${crop.deliveryWidthPx}`,
  ].join("/");

  return imageURL.replace("/image/upload/", `/image/upload/${transform}/`);
}

function humanizeAssetID(assetId: string): string {
  return assetId
    .split(/[_-]+/)
    .filter(Boolean)
    .map((segment) => segment[0]?.toUpperCase() + segment.slice(1))
    .join(" ");
}

function deterministicAvatarSlot(role: string, entityId: string, candidates: readonly number[]): number {
  if (candidates.length === 0) {
    return 0;
  }

  const bytes = new TextEncoder().encode(`avatar-slot-v1\u0000${role}\u0000${entityId}`);
  let hash = 0xcbf29ce484222325n;
  const prime = 0x100000001b3n;

  for (const byte of bytes) {
    hash ^= BigInt(byte);
    hash = BigInt.asUintN(64, hash * prime);
  }

  return candidates[Number(hash % BigInt(candidates.length))] ?? 0;
}
