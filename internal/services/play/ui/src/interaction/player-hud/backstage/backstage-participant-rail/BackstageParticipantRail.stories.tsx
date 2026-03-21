import type { Meta, StoryObj } from "@storybook/react-vite";
import { BackstageParticipantRail } from "./BackstageParticipantRail";
import { backstageFixtureCatalog } from "./fixtures";

const meta = {
  title: "Interaction/Player HUD/Backstage/Participant Rail",
  component: BackstageParticipantRail,
  parameters: {
    docs: {
      description: {
        component:
          "Right-side portrait rail for Backstage, with tooltip-backed typing and ready overlays plus a GM-authority marker.",
      },
    },
  },
  decorators: [
    (Story) => (
      <div className="flex h-dvh justify-end bg-base-100">
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof BackstageParticipantRail>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Discussion: Story = {
  args: {
    participants: backstageFixtureCatalog.openDiscussion.participants,
    viewerParticipantId: backstageFixtureCatalog.openDiscussion.viewerParticipantId,
    gmAuthorityParticipantId: backstageFixtureCatalog.openDiscussion.gmAuthorityParticipantId,
  },
  parameters: {
    docs: {
      description: {
        story:
          "Backstage discussion state, showing the viewer portrait, idle participants, a typing overlay for the participant currently composing an OOC note, and the GM authority owner.",
      },
    },
  },
};

export const WaitingOnGM: Story = {
  args: {
    participants: backstageFixtureCatalog.waitingOnGM.participants,
    viewerParticipantId: backstageFixtureCatalog.waitingOnGM.viewerParticipantId,
    gmAuthorityParticipantId: backstageFixtureCatalog.waitingOnGM.gmAuthorityParticipantId,
  },
  parameters: {
    docs: {
      description: {
        story:
          "All player portraits are marked ready while the GM remains in a neutral waiting state, with the authority icon showing who can resume on-stage play.",
      },
    },
  },
};
