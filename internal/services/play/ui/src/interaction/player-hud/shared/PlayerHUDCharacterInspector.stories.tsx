import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import {
  playerHUDCharacterCatalog,
  playerHUDCharacterInspectionCatalog,
} from "./character-inspection-fixtures";
import { PlayerHUDCharacterInspectorDialog } from "./PlayerHUDCharacterInspector";

const characterReferenceByID = Object.values(playerHUDCharacterCatalog).reduce<
  Record<string, (typeof playerHUDCharacterCatalog)[keyof typeof playerHUDCharacterCatalog]>
>((accumulator, character) => {
  accumulator[character.id] = character;
  return accumulator;
}, {});

function PlayerHUDCharacterInspectorPreview(input: {
  participantName: string;
  characterIDs: string[];
  activeCharacterId?: string;
  isViewer?: boolean;
}) {
  const [activeCharacterId, setActiveCharacterID] = useState(
    input.activeCharacterId,
  );

  return (
    <PlayerHUDCharacterInspectorDialog
      isOpen
      participantName={input.participantName}
      characters={input.characterIDs.map((characterID) => ({
        id: characterID,
        name: characterReferenceByID[characterID]?.name ?? characterID,
        avatarUrl: characterReferenceByID[characterID]?.avatarUrl,
      }))}
      activeCharacterId={activeCharacterId}
      isViewer={input.isViewer}
      characterInspectionCatalog={playerHUDCharacterInspectionCatalog}
      onCharacterChange={setActiveCharacterID}
      onClose={() => {}}
    />
  );
}

const meta = {
  title: "Interaction/Player HUD/Shared/Character Inspector",
  component: PlayerHUDCharacterInspectorPreview,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Shared modal inspector that renders the Daggerheart full card first, supports multi-character switching, and can toggle into the full character sheet.",
      },
    },
  },
  decorators: [
    (Story) => (
      <div className="min-h-dvh bg-base-100 p-6">
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof PlayerHUDCharacterInspectorPreview>;

export default meta;

type Story = StoryObj<typeof meta>;

export const MultiCharacter: Story = {
  args: {
    participantName: "Rhea",
    characterIDs: [
      playerHUDCharacterCatalog.aria.id,
      playerHUDCharacterCatalog.sable.id,
      playerHUDCharacterCatalog.mira.id,
      playerHUDCharacterCatalog.rowan.id,
    ],
    activeCharacterId: playerHUDCharacterCatalog.aria.id,
    isViewer: true,
  },
};

export const NoCharacters: Story = {
  args: {
    participantName: "Guide",
    characterIDs: [],
  },
};
