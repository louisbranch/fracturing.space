import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { BackstageCompose } from "./BackstageCompose";

function BackstageComposePreview({
  disabled = false,
  viewerReady = false,
}: {
  disabled?: boolean;
  viewerReady?: boolean;
}) {
  const [draft, setDraft] = useState("");
  const [ready, setReady] = useState(viewerReady);
  return (
    <BackstageCompose
      draft={draft}
      viewerReady={ready}
      disabled={disabled}
      onDraftChange={setDraft}
      onSend={() => setDraft("")}
      onReadyToggle={() => setReady((current) => !current)}
    />
  );
}

const meta = {
  title: "Interaction/Player HUD/Backstage/Compose",
  component: BackstageComposePreview,
  parameters: {
    docs: {
      description: {
        component: "Backstage-owned compose surface for posting OOC notes and managing player readiness without reusing the Side Chat composer.",
      },
    },
  },
  decorators: [
    (Story) => (
      <div className="flex h-48 w-[28rem] flex-col justify-end">
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

export const ViewerReady: Story = {
  args: { viewerReady: true },
};
