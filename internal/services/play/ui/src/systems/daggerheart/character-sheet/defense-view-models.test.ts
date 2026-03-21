import { describe, expect, it } from "vitest";
import {
  armorDisplayModel,
  ARMOR_GRID_SIZE,
  evasionDisplayModel,
} from "./defense-view-models";

describe("character sheet defense view models", () => {
  it("maps a partial armor track into filled, spent, and muted shield counts", () => {
    const model = armorDisplayModel({ current: 4, max: 5 });

    expect(model).toMatchObject({
      value: 5,
      filledCount: 4,
      spentCount: 1,
      mutedCount: ARMOR_GRID_SIZE - 5,
      summary: "4 filled, 1 spent, 7 unavailable",
    });
    expect(model?.cells.map((cell) => cell.state)).toEqual([
      "filled",
      "filled",
      "filled",
      "filled",
      "spent",
      "muted",
      "muted",
      "muted",
      "muted",
      "muted",
      "muted",
      "muted",
    ]);
  });

  it("clamps malformed armor values into the fixed 12-slot range", () => {
    const model = armorDisplayModel({ current: 15, max: 20 });

    expect(model).toMatchObject({
      value: ARMOR_GRID_SIZE,
      filledCount: ARMOR_GRID_SIZE,
      spentCount: 0,
      mutedCount: 0,
    });

    const empty = armorDisplayModel({ current: -2, max: -1 });
    expect(empty).toMatchObject({
      value: 0,
      filledCount: 0,
      spentCount: 0,
      mutedCount: ARMOR_GRID_SIZE,
    });
  });

  it("normalizes evasion into a non-negative integer for the large stat display", () => {
    expect(evasionDisplayModel(10.9)).toEqual({ value: 10 });
    expect(evasionDisplayModel(-3)).toEqual({ value: 0 });
    expect(evasionDisplayModel(undefined)).toBeUndefined();
  });
});
