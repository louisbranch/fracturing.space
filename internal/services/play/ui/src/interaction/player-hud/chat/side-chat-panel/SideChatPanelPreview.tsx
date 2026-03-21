import { useState } from "react";
import type { SideChatState } from "../../shared/contract";
import { SideChatPanel } from "./SideChatPanel";

type SideChatPanelPreviewProps = {
  initialState: SideChatState;
};

// SideChatPanelPreview wraps SideChatPanel with local draft state for
// interactive Storybook usage.
export function SideChatPanelPreview({ initialState }: SideChatPanelPreviewProps) {
  const [draft, setDraft] = useState("");

  return (
    <SideChatPanel
      state={initialState}
      draft={draft}
      onDraftChange={setDraft}
      onSend={() => setDraft("")}
    />
  );
}
