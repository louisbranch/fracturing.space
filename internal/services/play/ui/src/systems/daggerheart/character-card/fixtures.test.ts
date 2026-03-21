import { describe, expect, it } from "vitest";
import { characterAvatarPreviewAssets } from "../../../storybook/preview-assets/fixtures";
import { characterCardFixtures } from "./fixtures";

describe("character card fixtures", () => {
  it("uses shared character preview assets for non-placeholder portraits", () => {
    expect(characterCardFixtures.full.portrait.src).toBe(characterAvatarPreviewAssets[0]?.imageUrl);
    expect(characterCardFixtures.minimal.portrait.src).toBe(characterAvatarPreviewAssets[1]?.imageUrl);
    expect(characterCardFixtures.partial.portrait.src).toBeUndefined();
  });
});
