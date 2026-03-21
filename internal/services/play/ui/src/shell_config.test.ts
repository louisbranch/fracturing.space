import { describe, expect, it } from "vitest";
import { parseShellConfigScript, readShellConfig } from "./shell_config";

describe("shell_config", () => {
  it("parses injected shell config JSON", () => {
    expect(
      parseShellConfigScript({
        textContent: JSON.stringify({
          campaign_id: " c1 ",
          bootstrap_path: " /api/campaigns/c1/bootstrap ",
          realtime_path: " /realtime ",
          back_url: " http://example.com/app/campaigns/c1 ",
        }),
      }),
    ).toEqual({
      campaignId: "c1",
      bootstrapPath: "/api/campaigns/c1/bootstrap",
      realtimePath: "/realtime",
      backURL: "http://example.com/app/campaigns/c1",
    });
  });

  it("returns null for malformed shell config JSON", () => {
    expect(parseShellConfigScript({ textContent: "{oops" })).toBeNull();
  });

  it("returns null when the shell config element is absent", () => {
    expect(
      readShellConfig({
        getElementById: () => null,
      }),
    ).toBeNull();
  });
});
