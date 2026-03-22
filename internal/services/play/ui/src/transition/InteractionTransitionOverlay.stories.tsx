import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { InteractionTransitionOverlay } from "./InteractionTransitionOverlay";
import { useTransitionAudio } from "./useTransitionAudio";

const INTERACTION_SFX_URL =
  "https://res.cloudinary.com/fracturing-space/video/upload/v1774197569/high_fantasy/interface_sound_effect/v1/scene_interaction_transition_page_turn.mp3";

type PulseArgs = {
  durationMs: number;
  blurPx: number;
  spreadPx: number;
  themeColor: string;
  opacity: number;
  sound: boolean;
};

function InteractionTransitionOverlayPreview({
  durationMs,
  blurPx,
  spreadPx,
  themeColor,
  opacity,
  sound,
}: PulseArgs) {
  const [active, setActive] = useState(false);
  const { playInteractionSound } = useTransitionAudio({
    interactionUrl: INTERACTION_SFX_URL,
  });

  const style = {
    "--pulse-duration": `${durationMs}ms`,
    "--pulse-blur": `${blurPx}px`,
    "--pulse-spread": `${spreadPx}px`,
    "--pulse-color": `color-mix(in oklch, var(--color-${themeColor}) ${Math.round(opacity * 100)}%, transparent)`,
  } as React.CSSProperties;

  return (
    <div className="flex h-64 items-center justify-center bg-base-200" style={style}>
      <div className="flex flex-col items-center gap-4">
        <button
          type="button"
          className="btn btn-primary"
          onClick={() => {
            setActive(true);
            if (sound) playInteractionSound();
          }}
        >
          Trigger interaction pulse
        </button>
        <InteractionTransitionOverlay
          active={active}
          onTransitionEnd={() => setActive(false)}
          fallbackMs={durationMs + 100}
        >
          <div className="card bg-base-300 p-6 text-center shadow-lg">
            <p className="font-display text-lg">GM Interaction Card</p>
            <p className="text-sm opacity-60">Placeholder content</p>
          </div>
        </InteractionTransitionOverlay>
      </div>
    </div>
  );
}

const meta = {
  title: "Interaction/Player HUD/FX/Interaction Transition Overlay",
  component: InteractionTransitionOverlayPreview,
  tags: ["autodocs"],
  args: {
    durationMs: 2000,
    blurPx: 12,
    spreadPx: 2,
    themeColor: "primary",
    opacity: 0.45,
    sound: true,
  },
  argTypes: {
    durationMs: {
      control: { type: "range", min: 100, max: 2000, step: 50 },
    },
    blurPx: {
      control: { type: "range", min: 0, max: 40, step: 1 },
    },
    spreadPx: {
      control: { type: "range", min: 0, max: 10, step: 1 },
    },
    themeColor: {
      control: "select",
      options: ["primary", "secondary", "accent", "warning", "error", "success", "info"],
    },
    opacity: {
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
          "Wrapper that applies an interaction-pulse animation to its children when a new GM interaction arrives. Use the controls to experiment with animation parameters.",
      },
    },
  },
} satisfies Meta<typeof InteractionTransitionOverlayPreview>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};
