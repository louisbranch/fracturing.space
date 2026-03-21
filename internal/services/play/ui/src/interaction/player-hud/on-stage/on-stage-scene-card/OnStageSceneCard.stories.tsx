import type { Meta, StoryObj } from "@storybook/react-vite";
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
          "Scene overview card for On Stage, carrying the active scene, scene portraits, and the collapsible scene description.",
      },
    },
  },
} satisfies Meta<typeof OnStageSceneCard>;

export default meta;

type Story = StoryObj<typeof meta>;

export const OpeningScene: Story = {
  args: {
    sceneName: onStageFixtureCatalog.actingEmpty.scene.name,
    sceneDescription: onStageFixtureCatalog.actingEmpty.scene.description,
    sceneCharacters: onStageFixtureCatalog.actingEmpty.scene.characters,
    resolvedInteractionCount: onStageFixtureCatalog.actingEmpty.scene.resolvedInteractionCount,
    expanded: true,
    onToggle: () => {},
  },
};

export const CollapsedAfterFirstResolution: Story = {
  args: {
    sceneName: onStageFixtureCatalog.viewerPosted.scene.name,
    sceneDescription: onStageFixtureCatalog.viewerPosted.scene.description,
    sceneCharacters: onStageFixtureCatalog.viewerPosted.scene.characters,
    resolvedInteractionCount: onStageFixtureCatalog.viewerPosted.scene.resolvedInteractionCount,
    expanded: false,
    onToggle: () => {},
  },
};

const onStageAllStates = [
  { name: "Opening", state: onStageFixtureCatalog.actingEmpty, expanded: true },
  { name: "Collapsed", state: onStageFixtureCatalog.viewerPosted, expanded: false },
] as const;

export const AllStates: Story = {
  args: {
    sceneName: onStageFixtureCatalog.viewerPosted.scene.name,
    sceneDescription: onStageFixtureCatalog.viewerPosted.scene.description,
    sceneCharacters: onStageFixtureCatalog.viewerPosted.scene.characters,
    resolvedInteractionCount: onStageFixtureCatalog.viewerPosted.scene.resolvedInteractionCount,
    expanded: false,
    onToggle: () => {},
  },
  render: () => (
    <div className="flex max-w-4xl flex-col gap-4">
      {onStageAllStates.map(({ name, state, expanded }) => (
        <div key={name} className="flex flex-col gap-2">
          <div className="preview-kicker">{name}</div>
          <OnStageSceneCard
            sceneName={state.scene.name}
            sceneDescription={state.scene.description}
            sceneCharacters={state.scene.characters}
            resolvedInteractionCount={state.scene.resolvedInteractionCount}
            expanded={expanded}
            onToggle={() => {}}
          />
        </div>
      ))}
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: "Comparison of the default expanded opening scene and the collapsed state after the first resolved interaction.",
      },
    },
  },
};
