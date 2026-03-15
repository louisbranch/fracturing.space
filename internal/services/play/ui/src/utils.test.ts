import { describe, expect, it } from "vitest";
import { formatSessionLabel } from "./view_models";
import { mergeMessages, normalizeSnapshot, resolveCampaignId } from "./utils";

describe("resolveCampaignId", () => {
  it("extracts the campaign identifier from the play route", () => {
    expect(resolveCampaignId("/campaigns/camp-123")).toBe("camp-123");
  });
});

describe("mergeMessages", () => {
  it("keeps one message per id and sorts by sequence", () => {
    const merged = mergeMessages(
      [
        {
          message_id: "msg-2",
          campaign_id: "camp-1",
          session_id: "sess-1",
          sequence_id: 2,
          sent_at: "",
          actor: { participant_id: "part-2", name: "Mira" },
          body: "Second",
        },
      ],
      [
        {
          message_id: "msg-1",
          campaign_id: "camp-1",
          session_id: "sess-1",
          sequence_id: 1,
          sent_at: "",
          actor: { participant_id: "part-1", name: "Thorn" },
          body: "First",
        },
        {
          message_id: "msg-2",
          campaign_id: "camp-1",
          session_id: "sess-1",
          sequence_id: 2,
          sent_at: "",
          actor: { participant_id: "part-2", name: "Mira" },
          body: "Second",
        },
      ],
    );

    expect(merged.map((message) => message.sequence_id)).toEqual([1, 2]);
  });
});

describe("formatSessionLabel", () => {
  it("returns a fallback label only when no active session exists", () => {
    expect(formatSessionLabel()).toBe("No active session");
    expect(formatSessionLabel({ session_id: "sess-1", name: "" })).toBe("Untitled session");
    expect(formatSessionLabel({ session_id: "sess-1", name: "The Old Man" })).toBe("The Old Man");
  });
});

describe("normalizeSnapshot", () => {
  it("fills missing repeated fields from realtime payloads", () => {
    const snapshot = normalizeSnapshot(
      {
        campaign_id: "camp-1",
        interaction_state: {
          campaign_id: "camp-1",
          campaign_name: "Guildhouse",
          gm_authority_participant_id: "gm-1",
        },
        system: { id: "daggerheart", version: "1.0.0", name: "Daggerheart" },
        chat: { session_id: "sess-1", latest_sequence_id: 2, messages: [], history_url: "/history" },
        realtime: { url: "/realtime", protocol_version: 1 },
      },
      {
        interaction_state: {
          campaign_id: "camp-1",
          campaign_name: "Guildhouse",
          gm_authority_participant_id: "gm-1",
          player_phase: {
            phase_id: "phase-1",
            status: 2,
            frame_text: "",
            acting_character_ids: undefined as never,
            acting_participant_ids: undefined as never,
            slots: [
              {
                participant_id: "player-1",
                summary_text: "",
                character_ids: undefined as never,
                yielded: false,
                review_status: 0,
                review_reason: "",
                review_character_ids: undefined as never,
              },
            ],
          },
          ooc: {
            open: true,
            posts: undefined as never,
            ready_to_resume_participant_ids: undefined as never,
          },
        },
        latest_game_sequence: 4,
        chat: { session_id: "sess-1", latest_sequence_id: 2, messages: [], history_url: "/history" },
      },
    );

    expect(snapshot.interaction_state.player_phase?.acting_character_ids).toEqual([]);
    expect(snapshot.interaction_state.player_phase?.acting_participant_ids).toEqual([]);
    expect(snapshot.interaction_state.player_phase?.slots[0]?.character_ids).toEqual([]);
    expect(snapshot.interaction_state.player_phase?.slots[0]?.review_character_ids).toEqual([]);
    expect(snapshot.interaction_state.ooc?.posts).toEqual([]);
    expect(snapshot.interaction_state.ooc?.ready_to_resume_participant_ids).toEqual([]);
  });
});
