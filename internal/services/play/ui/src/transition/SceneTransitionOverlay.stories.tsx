import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { SceneTransitionOverlay } from "./SceneTransitionOverlay";
import { useTransitionAudio } from "./useTransitionAudio";

const SCENE_SFX_URL =
  "https://res.cloudinary.com/fracturing-space/video/upload/v1774198004/high_fantasy/interface_sound_effect/v1/scene_transition_slide_bookshelf.mp3";

type FadeArgs = {
  durationMs: number;
  overlayOpacity: number;
  sound: boolean;
};

function SceneTransitionOverlayPreview({
  durationMs,
  overlayOpacity,
  sound,
}: FadeArgs) {
  const [key, setKey] = useState(0);
  const { playSceneSound } = useTransitionAudio({
    sceneUrl: SCENE_SFX_URL,
  });

  const style = {
    "--scene-fade-duration": `${durationMs}ms`,
    "--scene-fade-bg": `rgb(0 0 0 / ${overlayOpacity})`,
  } as React.CSSProperties;

  return (
    <div
      className="relative flex h-64 items-center justify-center bg-base-200"
      style={style}
    >
      <button
        type="button"
        className="btn btn-primary"
        onClick={() => {
          setKey((k) => k + 1);
          if (sound) playSceneSound();
        }}
      >
        Trigger scene transition
      </button>
      <SceneTransitionOverlay transitionKey={key} fallbackMs={durationMs + 100} />
    </div>
  );
}

const meta = {
  title: "Interaction/Player HUD/FX/Scene Transition Overlay",
  component: SceneTransitionOverlayPreview,
  tags: ["autodocs"],
  args: {
    durationMs: 2000,
    overlayOpacity: 0.55,
    sound: true,
  },
  argTypes: {
    durationMs: {
      control: { type: "range", min: 100, max: 3000, step: 50 },
    },
    overlayOpacity: {
      control: { type: "range", min: 0, max: 1, step: 0.05 },
    },
    sound: {
      control: "boolean",
    },
  },
  parameters: {
    docs: {
      description: {
        component:
          "Full-viewport dark overlay that fades out, triggered by scene changes. Use the controls to experiment with duration, opacity, and sound.",
      },
    },
  },
} satisfies Meta<typeof SceneTransitionOverlayPreview>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};
