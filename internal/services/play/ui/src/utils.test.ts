import { describe, expect, it } from "vitest";
import { mergeMessages, resolveCampaignId, sessionLabel } from "./utils";

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

describe("sessionLabel", () => {
  it("returns a fallback label only when no active session exists", () => {
    expect(sessionLabel()).toBe("No active session");
    expect(sessionLabel({ session_id: "sess-1", name: "" })).toBe("Untitled session");
    expect(sessionLabel({ session_id: "sess-1", name: "The Old Man" })).toBe("The Old Man");
  });
});
