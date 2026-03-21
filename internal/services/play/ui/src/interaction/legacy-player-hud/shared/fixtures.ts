import type { PlayerComposerState, PlayerHUDState } from "./contract";

const defaultDrafts = {
  scratch: "We should force them toward the bridge anchor before the next gust hits.",
  scene: "Aria drops low, braces the rope line, and calls Corin forward before the scout slips.",
  ooc: "Rules check: does the ward react to tools or only direct contact?",
  chat: "Ready when you are.",
};

function baseComposer(): PlayerComposerState {
  return {
    activeMode: "scratch",
    minimized: false,
    drafts: { ...defaultDrafts },
    scene: {
      enabled: false,
      reason: "Waiting for the GM to hand the scene back to players.",
      yielded: false,
    },
    ooc: {
      open: false,
      helperText: "Pause the table to open out-of-character discussion.",
    },
  };
}

function baseHUD(): PlayerHUDState {
  return {
    campaignName: "Cliffside Rescue",
    backURL: "/app/campaigns/camp-cliffside",
    connection: "connected",
    stage: {
      title: "Storm Ledge",
      content: [],
      emptyMessage: "[empty for now]",
    },
    composer: baseComposer(),
  };
}

export const playerHUDFixtureCatalog: Record<
  "playerTurnParticipant" | "gmTurnWaiting" | "oocPaused" | "reconnecting" | "collapsedComposer" | "emptyStage",
  PlayerHUDState
> = {
  playerTurnParticipant: {
    ...baseHUD(),
    composer: {
      ...baseComposer(),
      activeMode: "scene",
      scene: {
        enabled: true,
        yielded: false,
      },
    },
  },
  gmTurnWaiting: {
    ...baseHUD(),
    composer: {
      ...baseComposer(),
      activeMode: "scratch",
      scene: {
        enabled: false,
        reason: "Waiting for the GM to hand the scene back to players.",
        yielded: false,
      },
    },
  },
  oocPaused: {
    ...baseHUD(),
    composer: {
      ...baseComposer(),
      activeMode: "ooc",
      scene: {
        enabled: false,
        reason: "Scene play is paused while out-of-character discussion is open.",
        yielded: false,
      },
      ooc: {
        open: true,
        helperText: "The table is paused. OOC messages can be posted until the scene resumes.",
      },
    },
  },
  reconnecting: {
    ...baseHUD(),
    connection: "reconnecting",
    composer: {
      ...baseComposer(),
      activeMode: "scratch",
    },
  },
  collapsedComposer: {
    ...baseHUD(),
    composer: {
      ...baseComposer(),
      activeMode: "chat",
      minimized: true,
    },
  },
  emptyStage: {
    ...baseHUD(),
    stage: {
      title: "Storm Ledge",
      content: [],
      emptyMessage: "[empty for now]",
    },
    composer: {
      ...baseComposer(),
      activeMode: "scratch",
    },
  },
};

export const playerHUDComponentFixtures = {
  header: {
    connected: playerHUDFixtureCatalog.playerTurnParticipant,
    reconnecting: playerHUDFixtureCatalog.reconnecting,
    disconnected: {
      ...playerHUDFixtureCatalog.playerTurnParticipant,
      connection: "disconnected" as const,
    },
  },
  stage: {
    default: playerHUDFixtureCatalog.playerTurnParticipant.stage,
    empty: playerHUDFixtureCatalog.emptyStage.stage,
    scrolling: {
      ...playerHUDFixtureCatalog.playerTurnParticipant.stage,
      content: [
        "The cliff path shudders beneath your boots while the storm tears at the rope line anchored above the ledge.",
        "A trapped scout is slipping below the lantern light. Future GM narration and prompts will render here when the stage is wired to runtime state.",
        "This long-content fixture exists only to validate that the viewport scrolls internally instead of expanding the entire page.",
      ],
    },
  },
  composer: {
    playerTurn: playerHUDFixtureCatalog.playerTurnParticipant.composer,
    gmTurn: playerHUDFixtureCatalog.gmTurnWaiting.composer,
    oocPaused: playerHUDFixtureCatalog.oocPaused.composer,
    collapsed: playerHUDFixtureCatalog.collapsedComposer.composer,
  },
  shell: {
    playerTurn: playerHUDFixtureCatalog.playerTurnParticipant,
    oocPaused: playerHUDFixtureCatalog.oocPaused,
    reconnecting: playerHUDFixtureCatalog.reconnecting,
    collapsed: playerHUDFixtureCatalog.collapsedComposer,
  },
};
