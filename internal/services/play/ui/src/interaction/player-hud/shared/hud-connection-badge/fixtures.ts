export const hudConnectionBadgeFixtures = {
  connected: {
    connectionState: "connected" as const,
  },
  reconnecting: {
    connectionState: "reconnecting" as const,
  },
  disconnected: {
    connectionState: "disconnected" as const,
  },
};
