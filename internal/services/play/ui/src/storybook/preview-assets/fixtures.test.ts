import avatarsCatalogJSON from "../../../../../../platform/assets/catalog/data/avatars.v1.json";
import campaignCoverCatalogJSON from "../../../../../../platform/assets/catalog/data/campaign_covers.v1.json";
import { describe, expect, it } from "vitest";
import {
  campaignCoverPreviewAssets,
  characterAvatarPreviewAssets,
  participantAvatarPreviewAssets,
  portraitSheetPreviewAssets,
} from "./fixtures";

type AssetSetJSON = {
  id: string;
  asset_ids: string[];
};

type AvatarCatalogJSON = {
  manifest: {
    sets: AssetSetJSON[];
  };
  sheets: Array<{
    set_id: string;
    width_px: number;
    height_px: number;
    portraits: Array<{
      slot: number;
      x: number;
      y: number;
      width_px: number;
      height_px: number;
    }>;
  }>;
};

type CampaignCoverCatalogJSON = {
  sets: AssetSetJSON[];
};

const avatarCatalog = avatarsCatalogJSON as AvatarCatalogJSON;
const campaignCoverCatalog = campaignCoverCatalogJSON as CampaignCoverCatalogJSON;

describe("preview asset fixtures", () => {
  it("preserves the checked-in people avatar order for avatar-based preview arrays", () => {
    const expectedAssetIDs = assetIDsForSet(avatarCatalog.manifest.sets, "avatar_set_v1");

    expect(participantAvatarPreviewAssets.map((asset) => asset.assetId)).toEqual(expectedAssetIDs);
    expect(characterAvatarPreviewAssets.map((asset) => asset.assetId)).toEqual(expectedAssetIDs);
    expect(portraitSheetPreviewAssets.map((asset) => asset.assetId)).toEqual(expectedAssetIDs);
  });

  it("preserves the checked-in campaign cover order", () => {
    const expectedAssetIDs = assetIDsForSet(campaignCoverCatalog.sets, "campaign_cover_set_v1");

    expect(campaignCoverPreviewAssets.map((asset) => asset.assetId)).toEqual(expectedAssetIDs);
  });

  it("excludes blank-avatar assets from the main preview arrays", () => {
    expect(participantAvatarPreviewAssets.every((asset) => !asset.assetId.startsWith("blank_"))).toBe(true);
    expect(characterAvatarPreviewAssets.every((asset) => !asset.assetId.startsWith("blank_"))).toBe(true);
    expect(portraitSheetPreviewAssets.every((asset) => !asset.assetId.startsWith("blank_"))).toBe(true);
  });

  it("resolves cloudinary delivery URLs for every preview asset", () => {
    const allAssets = [
      ...participantAvatarPreviewAssets,
      ...characterAvatarPreviewAssets,
      ...portraitSheetPreviewAssets,
      ...campaignCoverPreviewAssets,
    ];

    expect(
      allAssets.every((asset) =>
        asset.imageUrl.startsWith("https://res.cloudinary.com/fracturing-space/image/upload/"),
      ),
    ).toBe(true);
  });

  it("uses the sheet slot-1 crop for participant previews", () => {
    const slotOne = requiredPortrait(1);

    expect(new Set(participantAvatarPreviewAssets.map((asset) => asset.crop.slot))).toEqual(new Set([1]));
    expect(participantAvatarPreviewAssets[0]?.crop).toEqual({
      slot: 1,
      x: slotOne.x,
      y: slotOne.y,
      widthPx: slotOne.width_px,
      heightPx: slotOne.height_px,
      deliveryWidthPx: 384,
    });
    expect(participantAvatarPreviewAssets[0]?.imageUrl).toContain(
      `c_crop,w_${slotOne.width_px},h_${slotOne.height_px},x_${slotOne.x},y_${slotOne.y}`,
    );
  });

  it("keeps character preview crops deterministic and limited to slots two through four", () => {
    const firstSixSlots = characterAvatarPreviewAssets.slice(0, 6).map((asset) => asset.crop.slot);

    expect(firstSixSlots).toEqual([2, 4, 3, 4, 3, 2]);
    expect(characterAvatarPreviewAssets.every((asset) => [2, 3, 4].includes(asset.crop.slot))).toBe(true);
  });

  it("keeps portrait-sheet previews uncropped at the avatar-sheet dimensions", () => {
    const sheet = requiredAvatarSheet();

    expect(portraitSheetPreviewAssets[0]?.imageUrl).not.toContain("c_crop,w_");
    expect(portraitSheetPreviewAssets.every((asset) => asset.widthPx === sheet.width_px)).toBe(true);
    expect(portraitSheetPreviewAssets.every((asset) => asset.heightPx === sheet.height_px)).toBe(true);
    expect(portraitSheetPreviewAssets[0]).toMatchObject({
      widthPx: sheet.width_px,
      heightPx: sheet.height_px,
    });
  });
});

function assetIDsForSet(sets: AssetSetJSON[], setId: string): string[] {
  const assetSet = sets.find((entry) => entry.id === setId);
  expect(assetSet).toBeDefined();
  return [...(assetSet?.asset_ids ?? [])].filter((assetId) => !assetId.startsWith("blank_"));
}

function requiredAvatarSheet() {
  const sheet = avatarCatalog.sheets.find((entry) => entry.set_id === "avatar_set_v1");
  expect(sheet).toBeDefined();
  return sheet!;
}

function requiredPortrait(slot: number) {
  const portrait = requiredAvatarSheet().portraits.find((entry) => entry.slot === slot);
  expect(portrait).toBeDefined();
  return portrait!;
}
