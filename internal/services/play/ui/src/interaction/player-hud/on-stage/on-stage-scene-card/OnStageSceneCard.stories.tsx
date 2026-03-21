import type { Meta, StoryObj } from "@storybook/react-vite";
import { onStageStatusBadge } from "../../shared/view-models";
import { OnStageSceneCard } from "./OnStageSceneCard";
import { onStageFixtureCatalog } from "./fixtures";

const meta = {
  title: "Interaction/Player HUD/On Stage/Scene Card",
  component: OnStageSceneCard,
  parameters: {
    docs: {
      description: {
        component:
          "Scene context card for On Stage, carrying the active scene, current GM output, current frame, and the acting roster for the beat.",
      },
    },
  },
} satisfies Meta<typeof OnStageSceneCard>;

export default meta;

type Story = StoryObj<typeof meta>;

export const FramedBeat: Story = {
  args: {
    sceneName: onStageFixtureCatalog.viewerPosted.sceneName,
    sceneDescription: onStageFixtureCatalog.viewerPosted.sceneDescription,
    gmOutputText: onStageFixtureCatalog.viewerPosted.gmOutputText,
    frameText: onStageFixtureCatalog.viewerPosted.frameText,
    actingCharacterNames: onStageFixtureCatalog.viewerPosted.actingCharacterNames,
    status: onStageStatusBadge(onStageFixtureCatalog.viewerPosted),
  },
};

export const WaitingOnGM: Story = {
  args: {
    sceneName: onStageFixtureCatalog.waitingOnGM.sceneName,
    sceneDescription: onStageFixtureCatalog.waitingOnGM.sceneDescription,
    gmOutputText: onStageFixtureCatalog.waitingOnGM.gmOutputText,
    frameText: onStageFixtureCatalog.waitingOnGM.frameText,
    actingCharacterNames: onStageFixtureCatalog.waitingOnGM.actingCharacterNames,
    status: onStageStatusBadge(onStageFixtureCatalog.waitingOnGM),
  },
};
