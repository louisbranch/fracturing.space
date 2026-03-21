import type { Meta, StoryObj } from "@storybook/react-vite";
import { BackstageOOCList } from "./BackstageOOCList";
import { backstageFixtureCatalog } from "./fixtures";

const meta = {
  title: "Interaction/Player HUD/Backstage/OOC List",
  component: BackstageOOCList,
  parameters: {
    docs: {
      description: {
        component: "Transcript list for authoritative OOC discussion in Backstage, reusing the shared chat grouping treatment with Backstage-specific labels.",
      },
    },
  },
  decorators: [
    (Story) => (
      <div className="flex h-dvh w-96 flex-col">
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof BackstageOOCList>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Empty: Story = {
  args: {
    messages: backstageFixtureCatalog.openEmpty.messages,
    participants: backstageFixtureCatalog.openEmpty.participants,
    viewerParticipantId: backstageFixtureCatalog.openEmpty.viewerParticipantId,
  },
  parameters: {
    docs: {
      description: {
        story: "Freshly opened OOC state with no posts yet, useful for validating empty-thread spacing and copy.",
      },
    },
  },
};

export const Discussion: Story = {
  args: {
    messages: backstageFixtureCatalog.openDiscussion.messages,
    participants: backstageFixtureCatalog.openDiscussion.participants,
    viewerParticipantId: backstageFixtureCatalog.openDiscussion.viewerParticipantId,
  },
  parameters: {
    docs: {
      description: {
        story: "Active OOC conversation between players and GM, showing grouped messages and viewer alignment inside the Backstage transcript.",
      },
    },
  },
};
