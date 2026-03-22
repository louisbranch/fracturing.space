import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { TransitionPreferencesProvider } from "./TransitionPreferencesContext";
import { TransitionSettingsModal } from "./TransitionSettingsModal";

function TransitionSettingsModalPreview() {
  const [open, setOpen] = useState(true);

  return (
    <TransitionPreferencesProvider>
      <div className="flex h-64 items-center justify-center">
        <button
          type="button"
          className="btn btn-primary"
          onClick={() => setOpen(true)}
        >
          Open Settings
        </button>
        <TransitionSettingsModal isOpen={open} onClose={() => setOpen(false)} />
      </div>
    </TransitionPreferencesProvider>
  );
}

const meta = {
  title: "Interaction/Player HUD/FX/Transition Settings Modal",
  component: TransitionSettingsModalPreview,
  tags: ["autodocs"],
  parameters: {
    docs: {
      description: {
        component:
          "Modal with toggle switches for controlling scene and interaction transition visual and audio effects.",
      },
    },
  },
} satisfies Meta<typeof TransitionSettingsModalPreview>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};
