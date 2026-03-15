import type { Meta, StoryObj } from "@storybook/react-vite";
import { OOCOverlayPanel } from "./OOCOverlayPanel";
import { oocOverlayPanelFixtures } from "./fixtures";

const meta = {
  title: "Interaction/Legacy/OOC Overlay Panel",
  component: OOCOverlayPanel,
  tags: ["autodocs"],
  parameters: {
    docs: {
      description: {
        component:
          "Explicit out-of-character pause surface with posts, ready-to-resume state, and resume affordance.",
      },
    },
  },
} satisfies Meta<typeof OOCOverlayPanel>;

export default meta;

type Story = StoryObj<typeof meta>;

export const ReadyToResume: Story = {
  args: {
    phase: oocOverlayPanelFixtures.phase,
    ooc: oocOverlayPanelFixtures.ooc,
  },
};
