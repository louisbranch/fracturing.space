import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { BaseGameView } from "./BaseGameView";
import { createSystemRenderViewModel } from "../../view_models";

describe("BaseGameView", () => {
  it("renders the current scene summary", () => {
    render(
      <BaseGameView
        bootstrap={{
          campaign_id: "camp-1",
          viewer: { participant_id: "p1", name: "Avery", role: "PLAYER" },
          system: { id: "daggerheart", version: "1.0.0", name: "Daggerheart" },
          interaction_state: {
            campaign_id: "camp-1",
            campaign_name: "The Guildhouse",
            active_scene: {
              scene_id: "scene-1",
              session_id: "sess-1",
              name: "Town Gate",
              description: "A tense arrival.",
              characters: [],
            },
            gm_authority_participant_id: "",
          },
          chat: { session_id: "sess-1", latest_sequence_id: 0, messages: [], history_url: "/history" },
          realtime: { url: "/realtime", protocol_version: 1 },
        }}
        snapshot={{
          interaction_state: {
            campaign_id: "camp-1",
            campaign_name: "The Guildhouse",
            active_scene: {
              scene_id: "scene-1",
              session_id: "sess-1",
              name: "Town Gate",
              description: "A tense arrival.",
              characters: [],
            },
            gm_authority_participant_id: "",
          },
          latest_game_sequence: 3,
          chat: { session_id: "sess-1", latest_sequence_id: 0, messages: [], history_url: "/history" },
        }}
        view={createSystemRenderViewModel({
          interaction_state: {
            campaign_id: "camp-1",
            campaign_name: "The Guildhouse",
            active_scene: {
              scene_id: "scene-1",
              session_id: "sess-1",
              name: "Town Gate",
              description: "A tense arrival.",
              characters: [],
            },
            gm_authority_participant_id: "",
          },
          latest_game_sequence: 3,
          chat: { session_id: "sess-1", latest_sequence_id: 0, messages: [], history_url: "/history" },
        })}
      />,
    );

    expect(screen.getByText("Town Gate")).toBeInTheDocument();
    expect(screen.getByText("A tense arrival.")).toBeInTheDocument();
  });

  it("renders unnamed active sessions as untitled instead of absent", () => {
    render(
      <BaseGameView
        bootstrap={{
          campaign_id: "camp-1",
          viewer: { participant_id: "p1", name: "Avery", role: "PLAYER" },
          system: { id: "daggerheart", version: "1.0.0", name: "Daggerheart" },
          interaction_state: {
            campaign_id: "camp-1",
            campaign_name: "The Guildhouse",
            active_session: {
              session_id: "sess-1",
              name: "",
            },
            gm_authority_participant_id: "",
          },
          chat: { session_id: "sess-1", latest_sequence_id: 0, messages: [], history_url: "/history" },
          realtime: { url: "/realtime", protocol_version: 1 },
        }}
        snapshot={{
          interaction_state: {
            campaign_id: "camp-1",
            campaign_name: "The Guildhouse",
            active_session: {
              session_id: "sess-1",
              name: "",
            },
            gm_authority_participant_id: "",
          },
          latest_game_sequence: 3,
          chat: { session_id: "sess-1", latest_sequence_id: 0, messages: [], history_url: "/history" },
        }}
        view={createSystemRenderViewModel({
          interaction_state: {
            campaign_id: "camp-1",
            campaign_name: "The Guildhouse",
            active_session: {
              session_id: "sess-1",
              name: "",
            },
            gm_authority_participant_id: "",
          },
          latest_game_sequence: 3,
          chat: { session_id: "sess-1", latest_sequence_id: 0, messages: [], history_url: "/history" },
        })}
      />,
    );

    expect(screen.getByText("Untitled session")).toBeInTheDocument();
  });
});
