import { describe, expect, it } from "vitest";
import { canonicalizeWindowLocation, resolveAppLocation } from "./app_mode";

describe("app mode resolution", () => {
  it("renders the root path as a Storybook handoff shell", () => {
    const resolution = resolveAppLocation({ pathname: "/", search: "" });

    expect(resolution).toEqual({
      kind: "render",
      mode: {
        kind: "root-placeholder",
      },
    });
  });

  it("treats the retired preview route as unsupported", () => {
    const resolution = resolveAppLocation({
      pathname: "/preview/character-card",
      search: "",
    });

    expect(resolution).toEqual({
      kind: "render",
      mode: {
        kind: "unsupported",
        path: "/preview/character-card",
      },
    });
  });

  it("keeps campaign routes on the runtime placeholder path", () => {
    const resolution = resolveAppLocation({
      pathname: "/campaigns/the-guildhouse",
      search: "",
    });

    expect(resolution).toEqual({
      kind: "render",
      mode: {
        kind: "runtime-placeholder",
        campaignId: "the-guildhouse",
      },
    });
  });

  it("treats unknown paths as unsupported", () => {
    const resolution = resolveAppLocation({ pathname: "/mystery", search: "" });

    expect(resolution).toEqual({
      kind: "render",
      mode: {
        kind: "unsupported",
        path: "/mystery",
      },
    });
  });

  it("keeps the current location unchanged when canonicalizing the live location", () => {
    const location = { pathname: "/", search: "" };
    const mode = canonicalizeWindowLocation(location);

    expect(mode).toEqual({
      kind: "root-placeholder",
    });
  });
});
