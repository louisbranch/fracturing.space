import type { Meta, StoryObj } from "@storybook/react-vite";
import {
  PlayerHUDCharacterInspectorDialog,
  usePlayerHUDCharacterInspector,
} from "../../shared/PlayerHUDCharacterInspector";
import { playerHUDCharacterInspectionCatalog } from "../../shared/character-inspection-fixtures";
import { OnStageCharacterAvatarStack } from "./OnStageCharacterAvatarStack";
import { onStageFixtureCatalog } from "./fixtures";

const multiCharacterParticipant =
  onStageFixtureCatalog.multiCharacterOwner.participants[0];

function OnStageCharacterAvatarStackPreview(
  args: React.ComponentProps<typeof OnStageCharacterAvatarStack>,
) {
  const { inspector, close, openForCharacter, setActiveCharacter } =
    usePlayerHUDCharacterInspector();

  return (
    <>
      <OnStageCharacterAvatarStack
        {...args}
        onCharacterInspect={(characterId) =>
          openForCharacter(
            {
              name: multiCharacterParticipant?.name ?? "Participant",
              characters: args.characters,
              isViewer: true,
            },
            characterId,
          )
        }
      />
      <PlayerHUDCharacterInspectorDialog
        isOpen={Boolean(inspector)}
        participantName={inspector?.participantName ?? ""}
        characters={inspector?.characters ?? []}
        activeCharacterId={inspector?.activeCharacterId}
        isViewer={inspector?.isViewer ?? false}
        characterInspectionCatalog={playerHUDCharacterInspectionCatalog}
        onCharacterChange={setActiveCharacter}
        onClose={close}
      />
    </>
  );
}

const meta = {
  title: "Interaction/Player HUD/On Stage/Character Avatar Stack",
  component: OnStageCharacterAvatarStackPreview,
  parameters: {
    docs: {
      description: {
        component:
          "Stacked character-avatar treatment for participant-owned On Stage slots, showing up to three portraits or two plus an ellipsis when more are involved.",
      },
    },
  },
  decorators: [
    (Story) => (
      <div className="flex min-h-32 items-center bg-base-100 px-6">
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof OnStageCharacterAvatarStackPreview>;

export default meta;

type Story = StoryObj<typeof meta>;

export const SingleCharacter: Story = {
  args: {
    characters: onStageFixtureCatalog.viewerPosted.slots[0]?.characters ?? [],
  },
};

export const Overflow: Story = {
  args: {
    characters: onStageFixtureCatalog.multiCharacterOwner.slots[0]?.characters ?? [],
  },
};
