import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { BackstageCompose } from "./BackstageCompose";

function BackstageComposePreview({ disabled = false }: { disabled?: boolean }) {
  const [draft, setDraft] = useState("");
  return (
    <BackstageCompose
      draft={draft}
      disabled={disabled}
      onDraftChange={setDraft}
      onSend={() => setDraft("")}
    />
  );
}

const meta = {
  title: "Interaction/Player HUD/Backstage/Compose",
  component: BackstageComposePreview,
  parameters: {
    docs: {
      description: {
        component: "Backstage-specific compose bar for posting OOC notes, clarifications, and table coordination messages.",
      },
    },
  },
  decorators: [
    (Story) => (
      <div className="flex h-32 w-96 flex-col justify-end">
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof BackstageComposePreview>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Enabled: Story = {};

export const Disabled: Story = {
  args: { disabled: true },
  parameters: {
    docs: {
      description: {
        story: "Closed or dormant state where OOC is not currently open, so the compose input is visible but unavailable.",
      },
    },
  },
};
