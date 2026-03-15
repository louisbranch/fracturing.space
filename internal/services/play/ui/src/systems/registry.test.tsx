import { describe, expect, it } from "vitest";
import { resolveSystemRenderer } from "./registry";

describe("resolveSystemRenderer", () => {
  it("selects the daggerheart adapter for daggerheart v1", () => {
    const renderer = resolveSystemRenderer("daggerheart", "v1");

    expect(renderer.id).toBe("daggerheart@v1");
  });

  it("falls back to the base renderer for unknown systems", () => {
    const renderer = resolveSystemRenderer("mystery", "alpha");

    expect(renderer.id).toBe("base");
  });
});
