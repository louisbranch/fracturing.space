import { describe, expect, it } from "vitest";
import { characterAvatarPreviewAssets } from "../../../storybook/preview-assets/fixtures";
import { characterSheetFixtures } from "./fixtures";

describe("character sheet fixtures", () => {
  it("reuses the shared character preview assets", () => {
    expect(characterSheetFixtures.full.portrait.src).toBe(characterAvatarPreviewAssets[0]?.imageUrl);
    expect(characterSheetFixtures.damaged.portrait.src).toBe(characterAvatarPreviewAssets[0]?.imageUrl);
  });
});
