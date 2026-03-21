import type { PlayerHUDState, SideChatMessage, SideChatParticipant, SideChatState } from "./contract";
import { participantAvatarPreviewAssets } from "../../../storybook/preview-assets/fixtures";

const [viewerAvatar, corinAvatar, gmAvatar] = participantAvatarPreviewAssets;

export const sideChatParticipants: SideChatParticipant[] = [
  { id: "p-viewer", name: "Aria", avatarUrl: viewerAvatar?.imageUrl },
  { id: "p-corin", name: "Corin", avatarUrl: corinAvatar?.imageUrl },
  { id: "p-gm", name: "GM Sable", avatarUrl: gmAvatar?.imageUrl },
];

export const sideChatMessages: SideChatMessage[] = [
  { id: "m1", participantId: "p-corin", body: "Ready when you are.", sentAt: "2026-03-18T16:30:00Z" },
  { id: "m2", participantId: "p-corin", body: "I'll take the left flank.", sentAt: "2026-03-18T16:30:15Z" },
  { id: "m3", participantId: "p-viewer", body: "Copy. Moving to the bridge.", sentAt: "2026-03-18T16:31:00Z" },
  { id: "m4", participantId: "p-gm", body: "Quick heads-up: I'm adding a weather complication next round.", sentAt: "2026-03-18T16:32:00Z" },
  { id: "m5", participantId: "p-viewer", body: "Sounds good!", sentAt: "2026-03-18T16:32:30Z" },
  { id: "m6", participantId: "p-viewer", body: "Should we prep anything?", sentAt: "2026-03-18T16:32:45Z" },
];

export const sideChatState: SideChatState = {
  viewerParticipantId: "p-viewer",
  participants: sideChatParticipants,
  messages: sideChatMessages,
};

export const emptySideChatState: SideChatState = {
  viewerParticipantId: "p-viewer",
  participants: sideChatParticipants,
  messages: [],
};

export const playerHUDFixtureCatalog: Record<
  "onStage" | "backstage" | "sideChat",
  PlayerHUDState
> = {
  onStage: { activeTab: "on-stage", sideChat: sideChatState },
  backstage: { activeTab: "backstage", sideChat: sideChatState },
  sideChat: { activeTab: "side-chat", sideChat: sideChatState },
};
