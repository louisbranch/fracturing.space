import { act, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { BootstrapResponse, WireRoomSnapshot } from "./api/types";
import { FrameType } from "./api/websocket";
import { PlayRuntime } from "./PlayRuntime";
import {
  playerHUDCharacterCatalog,
  playerHUDCharacterInspectionCatalog,
} from "./interaction/player-hud/shared/character-inspection-fixtures";

const fetchBootstrapMock = vi.fn<(path: string) => Promise<BootstrapResponse>>();
const fetchAIDebugTurnsMock = vi.fn();
const fetchAIDebugTurnMock = vi.fn();
const connectWebSocketMock = vi.fn();
const submitScenePlayerPostMock = vi.fn();
const yieldScenePlayerPhaseMock = vi.fn();
const unyieldScenePlayerPhaseMock = vi.fn();
const websocketConnection = {
  send: vi.fn(),
  close: vi.fn(),
};

vi.mock("./api/bootstrap", () => ({
  fetchBootstrap: (path: string) => fetchBootstrapMock(path),
}));

vi.mock("./api/aiDebug", () => ({
  fetchAIDebugTurns: (campaignId: string, pageToken?: string) => fetchAIDebugTurnsMock(campaignId, pageToken),
  fetchAIDebugTurn: (campaignId: string, turnId: string) => fetchAIDebugTurnMock(campaignId, turnId),
}));

vi.mock("./api/mutations", () => ({
  submitScenePlayerPost: (campaignId: string, body: unknown) => submitScenePlayerPostMock(campaignId, body),
  yieldScenePlayerPhase: (campaignId: string, body: unknown) => yieldScenePlayerPhaseMock(campaignId, body),
  unyieldScenePlayerPhase: (campaignId: string, body: unknown) => unyieldScenePlayerPhaseMock(campaignId, body),
  postSessionOOC: vi.fn(),
  markOOCReadyToResume: vi.fn(),
  clearOOCReadyToResume: vi.fn(),
}));

vi.mock("./api/websocket", async () => {
  const actual = await vi.importActual<typeof import("./api/websocket")>("./api/websocket");
  return {
    ...actual,
    connectWebSocket: (options: unknown) => {
      connectWebSocketMock(options);
      return websocketConnection;
    },
  };
});

function runtimeBootstrap(): BootstrapResponse {
  return {
    campaign_id: "c1",
    ai_debug_enabled: true,
    viewer: { participant_id: "p1", name: "Avery", role: "player" },
    system: { id: "daggerheart", version: "1.0", name: "Daggerheart" },
    interaction_state: {
      campaign_id: "c1",
      campaign_name: "The Guildhouse",
      viewer: { participant_id: "p1", name: "Avery", role: "player" },
      gm_authority_participant_id: "p2",
      active_session: { session_id: "s1", name: "Session 1" },
      active_scene: {
        scene_id: "sc1",
        name: "The Vault",
        characters: [{ character_id: playerHUDCharacterCatalog.aria.id, name: "Aria", owner_participant_id: "p1" }],
      },
      player_phase: {
        phase_id: "ph1",
        status: "players",
        acting_character_ids: [playerHUDCharacterCatalog.aria.id],
        acting_participant_ids: ["p1"],
        slots: [{
          participant_id: "p1",
          summary_text: "Aria braces the warded door.",
          character_ids: [playerHUDCharacterCatalog.aria.id],
          yielded: false,
          review_character_ids: [],
        }],
      },
    },
    participants: [
      { id: "p1", name: "Avery", role: "player", character_ids: [playerHUDCharacterCatalog.aria.id] },
      { id: "p2", name: "Guide", role: "gm", character_ids: [] },
    ],
    character_inspection_catalog: {
      [playerHUDCharacterCatalog.aria.id]: playerHUDCharacterInspectionCatalog[playerHUDCharacterCatalog.aria.id],
    },
    chat: {
      session_id: "s1",
      latest_sequence_id: 0,
      messages: [],
      history_url: "/api/campaigns/c1/chat/history",
    },
    realtime: { url: "/realtime", protocol_version: 1 },
  };
}

function runtimeBootstrapWithMultipleCharacters(): BootstrapResponse {
  return {
    ...runtimeBootstrap(),
    participants: [
      {
        id: "p1",
        name: "Avery",
        role: "player",
        character_ids: [playerHUDCharacterCatalog.aria.id, playerHUDCharacterCatalog.mira.id],
      },
      { id: "p2", name: "Guide", role: "gm", character_ids: [] },
    ],
    character_inspection_catalog: {
      [playerHUDCharacterCatalog.aria.id]: playerHUDCharacterInspectionCatalog[playerHUDCharacterCatalog.aria.id],
      [playerHUDCharacterCatalog.mira.id]: playerHUDCharacterInspectionCatalog[playerHUDCharacterCatalog.mira.id],
    },
  };
}

function runtimeSnapshot(): WireRoomSnapshot {
  const bootstrap = runtimeBootstrap();
  return {
    interaction_state: bootstrap.interaction_state,
    participants: bootstrap.participants,
    character_inspection_catalog: bootstrap.character_inspection_catalog,
    chat: bootstrap.chat,
    latest_game_sequence: 3,
  };
}

function runtimeSnapshotWithoutParticipantCharacters(): WireRoomSnapshot {
  const bootstrap = runtimeBootstrapWithMultipleCharacters();
  return {
    interaction_state: bootstrap.interaction_state,
    participants: [
      { id: "p1", name: "Avery", role: "player", character_ids: [] },
      { id: "p2", name: "Guide", role: "gm", character_ids: [] },
    ],
    character_inspection_catalog: bootstrap.character_inspection_catalog,
    chat: bootstrap.chat,
    latest_game_sequence: 3,
  };
}

function waitingOnGMBootstrap(): BootstrapResponse {
  const bootstrap = runtimeBootstrap();
  return {
    ...bootstrap,
    interaction_state: {
      ...bootstrap.interaction_state,
      player_phase: {
        ...bootstrap.interaction_state.player_phase!,
        status: "gm_review",
      },
    },
  };
}

function runtimeBootstrapWithAITurn(status: "queued" | "running" | "failed"): BootstrapResponse {
  const bootstrap = runtimeBootstrap();
  return {
    ...bootstrap,
    interaction_state: {
      ...bootstrap.interaction_state,
      ai_turn: {
        owner_participant_id: "p2",
        status,
      },
    },
  };
}

function switchedSceneSnapshot(): WireRoomSnapshot {
  const snapshot = runtimeSnapshot();
  return {
    ...snapshot,
    interaction_state: {
      ...snapshot.interaction_state,
      active_scene: {
        scene_id: "sc2",
        name: "The Observatory",
        characters: [{ character_id: playerHUDCharacterCatalog.aria.id, name: "Aria", owner_participant_id: "p1" }],
      },
    },
    latest_game_sequence: 4,
  };
}

describe("PlayRuntime", () => {
  beforeEach(() => {
    fetchBootstrapMock.mockReset();
    fetchAIDebugTurnsMock.mockReset();
    fetchAIDebugTurnsMock.mockResolvedValue({ turns: [], next_page_token: "" });
    fetchAIDebugTurnMock.mockReset();
    fetchAIDebugTurnMock.mockResolvedValue({ id: "turn-1", entries: [] });
    connectWebSocketMock.mockReset();
    submitScenePlayerPostMock.mockReset();
    submitScenePlayerPostMock.mockResolvedValue(runtimeSnapshot());
    yieldScenePlayerPhaseMock.mockReset();
    yieldScenePlayerPhaseMock.mockResolvedValue(runtimeSnapshot());
    unyieldScenePlayerPhaseMock.mockReset();
    unyieldScenePlayerPhaseMock.mockResolvedValue(runtimeSnapshot());
    websocketConnection.send.mockReset();
    websocketConnection.close.mockReset();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("opens the character inspector from the drawer and participant portrait after realtime ready", async () => {
    const user = userEvent.setup();
    fetchBootstrapMock.mockResolvedValue(runtimeBootstrap());

    render(
      <PlayRuntime
        shellConfig={{
          campaignId: "c1",
          bootstrapPath: "/api/campaigns/c1/bootstrap",
          realtimePath: "/realtime",
          backURL: "http://example.com/app/campaigns/c1",
        }}
      />,
    );

    await screen.findByLabelText("Player HUD shell");
    await waitFor(() => expect(connectWebSocketMock).toHaveBeenCalledTimes(1));

    const websocketOptions = connectWebSocketMock.mock.calls[0][0] as {
      onEvent: (event: { type: "ready"; snapshot: WireRoomSnapshot }) => void;
    };
    act(() => {
      websocketOptions.onEvent({ type: "ready", snapshot: runtimeSnapshot() });
    });

    expect(await screen.findByLabelText("On-stage participants")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Open campaign sidebar" }));
    await user.click(screen.getByRole("button", { name: "Characters" }));
    const drawer = screen.getByLabelText("Player HUD sidebar");
    expect(within(drawer).getByRole("link", { name: "Return to Campaign" })).toHaveAttribute("href", "http://example.com/app/campaigns/c1");
    await user.click(within(drawer).getByRole("button", { name: "Inspect Aria" }));

    let dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByRole("heading", { name: "Avery" })).toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: "Aria" })).toBeInTheDocument();

    await user.click(within(dialog).getByRole("button", { name: "Close character inspector" }));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());

    await user.click(screen.getByRole("button", { name: "Inspect Avery" }));
    dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByRole("heading", { name: "Avery" })).toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: "Aria" })).toBeInTheDocument();
  });

  it("uses campaign navigation characters when the participant rail snapshot omits character ids", async () => {
    const user = userEvent.setup();
    fetchBootstrapMock.mockResolvedValue(runtimeBootstrapWithMultipleCharacters());

    render(
      <PlayRuntime
        shellConfig={{
          campaignId: "c1",
          bootstrapPath: "/api/campaigns/c1/bootstrap",
          realtimePath: "/realtime",
          backURL: "http://example.com/app/campaigns/c1",
        }}
      />,
    );

    await screen.findByLabelText("Player HUD shell");
    await waitFor(() => expect(connectWebSocketMock).toHaveBeenCalledTimes(1));

    const websocketOptions = connectWebSocketMock.mock.calls[0][0] as {
      onEvent: (event: { type: "ready"; snapshot: WireRoomSnapshot }) => void;
    };
    act(() => {
      websocketOptions.onEvent({ type: "ready", snapshot: runtimeSnapshotWithoutParticipantCharacters() });
    });

    await user.click(await screen.findByRole("button", { name: "Inspect Avery" }));

    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByRole("heading", { name: "Avery" })).toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: "Aria" })).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Aria" })).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Mira" })).toBeInTheDocument();
  });

  it("submits on-stage actions with scene and character context", async () => {
    const user = userEvent.setup();
    fetchBootstrapMock.mockResolvedValue(runtimeBootstrap());

    render(
      <PlayRuntime
        shellConfig={{
          campaignId: "c1",
          bootstrapPath: "/api/campaigns/c1/bootstrap",
          realtimePath: "/realtime",
          backURL: "http://example.com/app/campaigns/c1",
        }}
      />,
    );

    await screen.findByLabelText("Player HUD shell");
    await user.type(screen.getByLabelText("On-stage action input"), "Aria forces the door.");
    await user.click(screen.getByRole("button", { name: "Submit" }));

    await waitFor(() => expect(submitScenePlayerPostMock).toHaveBeenCalledTimes(1));
    expect(submitScenePlayerPostMock).toHaveBeenCalledWith("c1", {
      scene_id: "sc1",
      character_ids: [playerHUDCharacterCatalog.aria.id],
      summary_text: "Aria forces the door.",
    });
    expect(yieldScenePlayerPhaseMock).not.toHaveBeenCalled();
  });

  it("submits and yields on-stage actions as sequential transitions", async () => {
    const user = userEvent.setup();
    fetchBootstrapMock.mockResolvedValue(runtimeBootstrap());

    render(
      <PlayRuntime
        shellConfig={{
          campaignId: "c1",
          bootstrapPath: "/api/campaigns/c1/bootstrap",
          realtimePath: "/realtime",
          backURL: "http://example.com/app/campaigns/c1",
        }}
      />,
    );

    await screen.findByLabelText("Player HUD shell");
    await user.type(screen.getByLabelText("On-stage action input"), "Aria forces the door.");
    await user.click(screen.getByRole("button", { name: "Submit & Yield" }));

    await waitFor(() => expect(submitScenePlayerPostMock).toHaveBeenCalledTimes(1));
    expect(submitScenePlayerPostMock).toHaveBeenCalledWith("c1", {
      scene_id: "sc1",
      character_ids: [playerHUDCharacterCatalog.aria.id],
      summary_text: "Aria forces the door.",
    });
    await waitFor(() => expect(yieldScenePlayerPhaseMock).toHaveBeenCalledTimes(1));
    expect(yieldScenePlayerPhaseMock).toHaveBeenCalledWith("c1", { scene_id: "sc1" });
  });

  it("yields the current scene with explicit scene context", async () => {
    const user = userEvent.setup();
    fetchBootstrapMock.mockResolvedValue(runtimeBootstrap());

    render(
      <PlayRuntime
        shellConfig={{
          campaignId: "c1",
          bootstrapPath: "/api/campaigns/c1/bootstrap",
          realtimePath: "/realtime",
          backURL: "http://example.com/app/campaigns/c1",
        }}
      />,
    );

    await screen.findByLabelText("Player HUD shell");
    await user.click(screen.getByRole("button", { name: "Yield" }));

    await waitFor(() => expect(yieldScenePlayerPhaseMock).toHaveBeenCalledTimes(1));
    expect(yieldScenePlayerPhaseMock).toHaveBeenCalledWith("c1", { scene_id: "sc1" });
  });

  it("resyncs the runtime after a 409 mutation conflict", async () => {
    const user = userEvent.setup();
    fetchBootstrapMock
      .mockResolvedValueOnce(runtimeBootstrap())
      .mockResolvedValueOnce(waitingOnGMBootstrap());
    submitScenePlayerPostMock.mockRejectedValue(Object.assign(new Error("action not allowed in current state"), {
      status: 409,
    }));

    render(
      <PlayRuntime
        shellConfig={{
          campaignId: "c1",
          bootstrapPath: "/api/campaigns/c1/bootstrap",
          realtimePath: "/realtime",
          backURL: "http://example.com/app/campaigns/c1",
        }}
      />,
    );

    await screen.findByLabelText("Player HUD shell");
    await user.type(screen.getByLabelText("On-stage action input"), "Aria forces the door.");
    await user.click(screen.getByRole("button", { name: "Submit" }));

    expect(await screen.findByText("Scene state changed. The play view was refreshed.")).toBeInTheDocument();
    await waitFor(() => expect(fetchBootstrapMock).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(screen.queryByRole("button", { name: "Submit" })).not.toBeInTheDocument());
  });

  it("updates the active scene when realtime interaction updates arrive", async () => {
    fetchBootstrapMock.mockResolvedValue(runtimeBootstrap());

    render(
      <PlayRuntime
        shellConfig={{
          campaignId: "c1",
          bootstrapPath: "/api/campaigns/c1/bootstrap",
          realtimePath: "/realtime",
          backURL: "http://example.com/app/campaigns/c1",
        }}
      />,
    );

    expect(await screen.findByText("The Vault")).toBeInTheDocument();
    await waitFor(() => expect(connectWebSocketMock).toHaveBeenCalledTimes(1));

    const websocketOptions = connectWebSocketMock.mock.calls[0][0] as {
      onEvent: (event: { type: "interaction.updated"; snapshot: WireRoomSnapshot }) => void;
    };
    act(() => {
      websocketOptions.onEvent({ type: "interaction.updated", snapshot: switchedSceneSnapshot() });
    });

    expect(await screen.findByText("The Observatory")).toBeInTheDocument();
    expect(screen.queryByText("The Vault")).not.toBeInTheDocument();
  });

  it("sends unified play.typing frames from on-stage and side-chat composers", async () => {
    const user = userEvent.setup();
    fetchBootstrapMock.mockResolvedValue(runtimeBootstrap());

    render(
      <PlayRuntime
        shellConfig={{
          campaignId: "c1",
          bootstrapPath: "/api/campaigns/c1/bootstrap",
          realtimePath: "/realtime",
          backURL: "http://example.com/app/campaigns/c1",
        }}
      />,
    );

    await screen.findByLabelText("Player HUD shell");
    await waitFor(() => expect(connectWebSocketMock).toHaveBeenCalledTimes(1));
    websocketConnection.send.mockClear();

    await user.type(screen.getByLabelText("On-stage action input"), "Aria");
    expect(websocketConnection.send).toHaveBeenCalledWith({
      type: FrameType.Typing,
      payload: { active: true },
    });

    await user.clear(screen.getByLabelText("On-stage action input"));
    expect(websocketConnection.send).toHaveBeenLastCalledWith({
      type: FrameType.Typing,
      payload: { active: false },
    });

    websocketConnection.send.mockClear();
    await user.click(screen.getByRole("button", { name: "Side Chat" }));
    await user.type(screen.getByLabelText("Chat message input"), "Ready");
    expect(websocketConnection.send).toHaveBeenCalledWith({
      type: FrameType.Typing,
      payload: { active: true },
    });

    await user.clear(screen.getByLabelText("Chat message input"));
    expect(websocketConnection.send).toHaveBeenLastCalledWith({
      type: FrameType.Typing,
      payload: { active: false },
    });
  });

  it("renders unified typing presence across on-stage, backstage, and side chat rails", async () => {
    const user = userEvent.setup();
    fetchBootstrapMock.mockResolvedValue(runtimeBootstrap());

    render(
      <PlayRuntime
        shellConfig={{
          campaignId: "c1",
          bootstrapPath: "/api/campaigns/c1/bootstrap",
          realtimePath: "/realtime",
          backURL: "http://example.com/app/campaigns/c1",
        }}
      />,
    );

    await screen.findByLabelText("Player HUD shell");
    await waitFor(() => expect(connectWebSocketMock).toHaveBeenCalledTimes(1));

    const websocketOptions = connectWebSocketMock.mock.calls[0][0] as {
      onEvent: (event: { type: "typing"; participantId: string; name: string; active: boolean }) => void;
    };
    act(() => {
      websocketOptions.onEvent({ type: "typing", participantId: "p2", name: "Guide", active: true });
    });

    expect(screen.getByLabelText("Guide: typing")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Backstage" }));
    expect(screen.getByLabelText("Guide: typing")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Side Chat" }));
    expect(screen.getByLabelText("Guide: typing")).toBeInTheDocument();
  });

  it("refreshes bootstrap when the server requests a resync", async () => {
    fetchBootstrapMock
      .mockResolvedValueOnce(runtimeBootstrap())
      .mockResolvedValueOnce({
        ...runtimeBootstrap(),
        interaction_state: {
          ...runtimeBootstrap().interaction_state,
          active_scene: {
            scene_id: "sc2",
            name: "The Observatory",
            characters: [{ character_id: playerHUDCharacterCatalog.aria.id, name: "Aria", owner_participant_id: "p1" }],
          },
        },
      });

    render(
      <PlayRuntime
        shellConfig={{
          campaignId: "c1",
          bootstrapPath: "/api/campaigns/c1/bootstrap",
          realtimePath: "/realtime",
          backURL: "http://example.com/app/campaigns/c1",
        }}
      />,
    );

    expect(await screen.findByText("The Vault")).toBeInTheDocument();
    await waitFor(() => expect(connectWebSocketMock).toHaveBeenCalledTimes(1));

    const websocketOptions = connectWebSocketMock.mock.calls[0][0] as {
      onEvent: (event: { type: "resync" }) => void;
    };
    act(() => {
      websocketOptions.onEvent({ type: "resync" });
    });

    await waitFor(() => expect(fetchBootstrapMock).toHaveBeenCalledTimes(2));
    expect(await screen.findByText("The Observatory")).toBeInTheDocument();
    expect(screen.queryByText("The Vault")).not.toBeInTheDocument();
  });

  it("renders live AI debug websocket updates without manual refresh", async () => {
    const user = userEvent.setup();
    fetchBootstrapMock.mockResolvedValue(runtimeBootstrap());
    fetchAIDebugTurnMock.mockResolvedValue({
      id: "turn-1",
      model: "gpt-5.4",
      status: "running",
      entry_count: 0,
      entries: [],
    });

    render(
      <PlayRuntime
        shellConfig={{
          campaignId: "c1",
          bootstrapPath: "/api/campaigns/c1/bootstrap",
          realtimePath: "/realtime",
          backURL: "http://example.com/app/campaigns/c1",
        }}
      />,
    );

    await screen.findByLabelText("Player HUD shell");
    await user.click(screen.getByRole("button", { name: "AI Debug" }));
    await waitFor(() => expect(fetchAIDebugTurnsMock).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(connectWebSocketMock).toHaveBeenCalledTimes(1));

    const websocketOptions = connectWebSocketMock.mock.calls[0][0] as {
      onEvent: (event: { type: "ai-debug.turn.updated"; update: unknown }) => void;
    };
    act(() => {
      websocketOptions.onEvent({
        type: "ai-debug.turn.updated",
        update: {
          turn: {
            id: "turn-1",
            model: "gpt-5.4",
            status: "running",
            started_at: "2026-03-22T12:00:00Z",
            updated_at: "2026-03-22T12:00:01Z",
            entry_count: 1,
          },
          appended_entries: [],
        },
      });
    });

    expect(await screen.findByText("gpt-5.4")).toBeInTheDocument();
    expect(screen.getByText("1 entries")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /gpt-5.4/i }));
    await waitFor(() => expect(fetchAIDebugTurnMock).toHaveBeenCalledWith("c1", "turn-1"));

    act(() => {
      websocketOptions.onEvent({
        type: "ai-debug.turn.updated",
        update: {
          turn: {
            id: "turn-1",
            model: "gpt-5.4",
            status: "running",
            started_at: "2026-03-22T12:00:00Z",
            updated_at: "2026-03-22T12:00:02Z",
            entry_count: 1,
          },
          appended_entries: [{
            sequence: 1,
            kind: "tool_call",
            tool_name: "scene_create",
            payload: "{\"name\":\"Harbor\"}",
            created_at: "2026-03-22T12:00:02Z",
          }],
        },
      });
    });

    expect(await screen.findByText("Tool call")).toBeInTheDocument();
    expect(screen.getByText("scene_create")).toBeInTheDocument();
  });

  it("shows AI status on the GM portrait across rails instead of on the AI Debug tab", async () => {
    const user = userEvent.setup();
    fetchBootstrapMock.mockResolvedValue(runtimeBootstrapWithAITurn("running"));

    render(
      <PlayRuntime
        shellConfig={{
          campaignId: "c1",
          bootstrapPath: "/api/campaigns/c1/bootstrap",
          realtimePath: "/realtime",
          backURL: "http://example.com/app/campaigns/c1",
        }}
      />,
    );

    await screen.findByLabelText("Player HUD shell");

    expect(screen.getByLabelText("Guide: AI thinking")).toBeInTheDocument();
    expect(screen.getByLabelText("Guide status: AI thinking")).toBeInTheDocument();
    expect(screen.getByLabelText("Guide GM authority")).toBeInTheDocument();

    // Invariant: AI status moved from the navbar badge to the GM portrait.
    expect(screen.getByRole("button", { name: "AI Debug" }).closest(".indicator")).toBeNull();

    await user.click(screen.getByRole("button", { name: "Backstage" }));
    expect(screen.getByLabelText("Guide: AI thinking")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Side Chat" }));
    expect(screen.getByLabelText("Guide: AI thinking")).toBeInTheDocument();
  });

  it("resyncs AI debug history after websocket reconnect once the panel was opened", async () => {
    const user = userEvent.setup();
    fetchBootstrapMock.mockResolvedValue(runtimeBootstrap());

    render(
      <PlayRuntime
        shellConfig={{
          campaignId: "c1",
          bootstrapPath: "/api/campaigns/c1/bootstrap",
          realtimePath: "/realtime",
          backURL: "http://example.com/app/campaigns/c1",
        }}
      />,
    );

    await screen.findByLabelText("Player HUD shell");
    await user.click(screen.getByRole("button", { name: "AI Debug" }));
    await waitFor(() => expect(fetchAIDebugTurnsMock).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(connectWebSocketMock).toHaveBeenCalledTimes(1));

    const websocketOptions = connectWebSocketMock.mock.calls[0][0] as {
      onEvent: (event: { type: "connection"; state: "reconnecting" | "connected" }) => void;
    };
    act(() => {
      websocketOptions.onEvent({ type: "connection", state: "reconnecting" });
      websocketOptions.onEvent({ type: "connection", state: "connected" });
    });

    await waitFor(() => expect(fetchAIDebugTurnsMock).toHaveBeenCalledTimes(2));
  });
});
