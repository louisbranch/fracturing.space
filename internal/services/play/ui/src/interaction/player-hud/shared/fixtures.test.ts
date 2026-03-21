import { describe, expect, it } from "vitest";
import {
  characterAvatarPreviewAssets,
  participantAvatarPreviewAssets,
} from "../../../storybook/preview-assets/fixtures";
import {
  backstageParticipants,
  onStageCharacterCatalog,
  onStageFixtureCatalog,
  onStageParticipants,
  playerHUDFixtureCatalog,
  sideChatParticipants,
} from "./fixtures";

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

  it("assigns preview avatars to every on-stage participant", () => {
    expect(onStageParticipants.map((participant) => participant.avatarUrl)).toEqual(
      participantAvatarPreviewAssets.slice(0, 3).map((asset) => asset.imageUrl),
    );
  });

  it("assigns preview avatars to on-stage character stacks", () => {
    const multiCharacterSlot = onStageFixtureCatalog.multiCharacterOwner.slots[0];
    expect(multiCharacterSlot?.characters.map((character) => character.avatarUrl)).toEqual(
      [
        onStageCharacterCatalog.aria.avatarUrl,
        onStageCharacterCatalog.sable.avatarUrl,
        onStageCharacterCatalog.mira.avatarUrl,
        onStageCharacterCatalog.rowan.avatarUrl,
      ],
    );
  });

  it("assigns a preview avatar to shell-only on-stage participants", () => {
    const shellOnlyParticipant = playerHUDFixtureCatalog.onStage.onStage.participants.find(
      (participant) => participant.id === "p-ives",
    );

    expect(shellOnlyParticipant?.avatarUrl).toEqual(participantAvatarPreviewAssets[3]?.imageUrl);
  });
});
