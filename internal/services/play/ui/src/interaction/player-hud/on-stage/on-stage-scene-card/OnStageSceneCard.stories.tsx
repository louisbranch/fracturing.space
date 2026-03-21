import type { Meta, StoryObj } from "@storybook/react-vite";
import { onStageStatusBadge } from "../../shared/view-models";
import { OnStageSceneCard } from "./OnStageSceneCard";
import { onStageFixtureCatalog } from "./fixtures";

const meta = {
  title: "Interaction/Player HUD/On Stage/Scene Card",
  component: OnStageSceneCard,
  tags: ["autodocs"],
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

const onStageAllStates = [
  { name: "Your Beat", state: onStageFixtureCatalog.viewerPosted },
  { name: "Yielded", state: onStageFixtureCatalog.yieldedWaiting },
  { name: "Changes Requested", state: onStageFixtureCatalog.changesRequested },
  { name: "OOC Open", state: onStageFixtureCatalog.oocBlocked },
  { name: "Waiting", state: onStageFixtureCatalog.waitingOnGM },
  { name: "AI Thinking", state: onStageFixtureCatalog.aiThinking },
  { name: "GM Delayed", state: onStageFixtureCatalog.aiFailed },
] as const;

export const AllStates: Story = {
  args: {
    sceneName: onStageFixtureCatalog.viewerPosted.sceneName,
    sceneDescription: onStageFixtureCatalog.viewerPosted.sceneDescription,
    gmOutputText: onStageFixtureCatalog.viewerPosted.gmOutputText,
    frameText: onStageFixtureCatalog.viewerPosted.frameText,
    actingCharacterNames: onStageFixtureCatalog.viewerPosted.actingCharacterNames,
    status: onStageStatusBadge(onStageFixtureCatalog.viewerPosted),
  },
  render: () => (
    <div className="flex max-w-4xl flex-col gap-4">
      {onStageAllStates.map(({ name, state }) => (
        <div key={name} className="flex flex-col gap-2">
          <div className="preview-kicker">{name}</div>
          <OnStageSceneCard
            sceneName={state.sceneName}
            sceneDescription={state.sceneDescription}
            gmOutputText={state.gmOutputText}
            frameText={state.frameText}
            actingCharacterNames={state.actingCharacterNames}
            status={onStageStatusBadge(state)}
          />
        </div>
      ))}
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: "Overview of every on-stage status badge state, including the loading-bar variants for pending GM-owned progress.",
      },
    },
  },
};
