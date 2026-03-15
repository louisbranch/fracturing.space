import { describe, expect, it } from "vitest";
import { resolveSystemRenderer } from "./registry";

describe("resolveSystemRenderer", () => {
  it("selects the daggerheart adapter for major version 1 systems", () => {
    const renderer = resolveSystemRenderer({ id: "daggerheart", version: "1.0.0" });

    expect(renderer.id).toBe("daggerheart@v1");
  });

  it("falls back to the base renderer for unknown systems", () => {
    const renderer = resolveSystemRenderer({ id: "mystery", version: "alpha" });

    expect(renderer.id).toBe("base");
  });
});
