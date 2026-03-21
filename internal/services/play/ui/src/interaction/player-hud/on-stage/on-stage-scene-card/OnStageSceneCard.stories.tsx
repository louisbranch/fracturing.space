import { onStageStatusDisplay } from "../shared/status-display";
import type { Meta, StoryObj } from "@storybook/react-vite";
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
    statusLabel: onStageStatusDisplay({
      mode: onStageFixtureCatalog.viewerPosted.mode,
      aiStatus: onStageFixtureCatalog.viewerPosted.aiStatus,
      disabledReason: onStageFixtureCatalog.viewerPosted.viewerControls.disabledReason,
      oocReason: onStageFixtureCatalog.viewerPosted.oocReason,
    }).badgeLabel,
    statusClassName: onStageStatusDisplay({
      mode: onStageFixtureCatalog.viewerPosted.mode,
      aiStatus: onStageFixtureCatalog.viewerPosted.aiStatus,
      disabledReason: onStageFixtureCatalog.viewerPosted.viewerControls.disabledReason,
      oocReason: onStageFixtureCatalog.viewerPosted.oocReason,
    }).badgeClassName,
    statusTooltip: onStageStatusDisplay({
      mode: onStageFixtureCatalog.viewerPosted.mode,
      aiStatus: onStageFixtureCatalog.viewerPosted.aiStatus,
      disabledReason: onStageFixtureCatalog.viewerPosted.viewerControls.disabledReason,
      oocReason: onStageFixtureCatalog.viewerPosted.oocReason,
    }).message,
  },
};

export const WaitingOnGM: Story = {
  args: {
    sceneName: onStageFixtureCatalog.waitingOnGM.sceneName,
    sceneDescription: onStageFixtureCatalog.waitingOnGM.sceneDescription,
    gmOutputText: onStageFixtureCatalog.waitingOnGM.gmOutputText,
    frameText: onStageFixtureCatalog.waitingOnGM.frameText,
    actingCharacterNames: onStageFixtureCatalog.waitingOnGM.actingCharacterNames,
    statusLabel: onStageStatusDisplay({
      mode: onStageFixtureCatalog.waitingOnGM.mode,
      aiStatus: onStageFixtureCatalog.waitingOnGM.aiStatus,
      disabledReason: onStageFixtureCatalog.waitingOnGM.viewerControls.disabledReason,
      oocReason: onStageFixtureCatalog.waitingOnGM.oocReason,
    }).badgeLabel,
    statusClassName: onStageStatusDisplay({
      mode: onStageFixtureCatalog.waitingOnGM.mode,
      aiStatus: onStageFixtureCatalog.waitingOnGM.aiStatus,
      disabledReason: onStageFixtureCatalog.waitingOnGM.viewerControls.disabledReason,
      oocReason: onStageFixtureCatalog.waitingOnGM.oocReason,
    }).badgeClassName,
    statusTooltip: onStageStatusDisplay({
      mode: onStageFixtureCatalog.waitingOnGM.mode,
      aiStatus: onStageFixtureCatalog.waitingOnGM.aiStatus,
      disabledReason: onStageFixtureCatalog.waitingOnGM.viewerControls.disabledReason,
      oocReason: onStageFixtureCatalog.waitingOnGM.oocReason,
    }).message,
  },
};
