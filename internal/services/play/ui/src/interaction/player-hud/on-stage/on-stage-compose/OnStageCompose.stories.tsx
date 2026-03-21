import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { OnStageCompose } from "./OnStageCompose";
import { onStageFixtureCatalog } from "./fixtures";

function OnStageComposePreview(input: {
  state: typeof onStageFixtureCatalog.viewerPosted;
  initialDraft?: string;
}) {
  const [draft, setDraft] = useState(input.initialDraft ?? "");

  return (
    <OnStageCompose
      draft={draft}
      controls={input.state.viewerControls}
      mechanicsExtension={input.state.mechanicsExtension}
      onDraftChange={setDraft}
      onSubmit={() => {}}
      onSubmitAndYield={() => {}}
      onYield={() => {}}
      onUnyield={() => {}}
    />
  );
}

const meta = {
  title: "Interaction/Player HUD/On Stage/Compose",
  component: OnStageComposePreview,
  parameters: {
    docs: {
      description: {
        component:
          "On Stage compose bar for participant-owned action summaries, paired with the stable mechanics extension seam and yield controls.",
      },
    },
  },
} satisfies Meta<typeof OnStageComposePreview>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Acting: Story = {
  args: {
    state: onStageFixtureCatalog.viewerPosted,
    initialDraft: "Aria hooks the pry tool into the seam.",
  },
};

export const Yielded: Story = {
  args: {
    state: onStageFixtureCatalog.yieldedWaiting,
    initialDraft: "Aria hooks the pry tool into the seam.",
  },
};

export const OOCBlocked: Story = {
  args: {
    state: onStageFixtureCatalog.oocBlocked,
    initialDraft: "",
  },
};
