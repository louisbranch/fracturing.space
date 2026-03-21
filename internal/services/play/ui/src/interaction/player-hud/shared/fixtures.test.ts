import { describe, expect, it } from "vitest";
import { participantAvatarPreviewAssets } from "../../../storybook/preview-assets/fixtures";
import { backstageParticipants, sideChatParticipants } from "./fixtures";

describe("player HUD shared fixtures", () => {
  it("assigns preview avatars to every side chat participant", () => {
    expect(sideChatParticipants.map((participant) => participant.avatarUrl)).toEqual(
      participantAvatarPreviewAssets.slice(0, 3).map((asset) => asset.imageUrl),
    );
  });

  it("assigns preview avatars to every backstage participant", () => {
    expect(backstageParticipants.map((participant) => participant.avatarUrl)).toEqual(
      participantAvatarPreviewAssets.slice(0, 3).map((asset) => asset.imageUrl),
    );
  });
});
