import type { DaggerheartTrackValue } from "../character-card/contract";

export const ARMOR_GRID_SIZE = 12;
export const ARMOR_GRID_COLUMNS = 3;

export type ArmorShieldState = "filled" | "spent" | "muted";

export type ArmorShieldCell = {
  index: number;
  state: ArmorShieldState;
};

export type ArmorDisplayModel = {
  value: number;
  filledCount: number;
  spentCount: number;
  mutedCount: number;
  cells: ArmorShieldCell[];
  summary: string;
};

export type EvasionDisplayModel = {
  value: number;
};

// armorDisplayModel normalizes armor track values into the fixed 12-slot shield
// display so the component can render state without mixing JSX and math.
export function armorDisplayModel(armor?: DaggerheartTrackValue): ArmorDisplayModel | undefined {
  if (!armor) {
    return undefined;
  }

  const value = clampCount(armor.max, ARMOR_GRID_SIZE);
  const filledCount = clampCount(armor.current, value);
  const spentCount = value - filledCount;
  const mutedCount = ARMOR_GRID_SIZE - value;
  const cells = Array.from({ length: ARMOR_GRID_SIZE }, (_, index) => ({
    index,
    state: shieldState(index, filledCount, spentCount),
  }));

  return {
    value,
    filledCount,
    spentCount,
    mutedCount,
    cells,
    summary: `${filledCount} filled, ${spentCount} spent, ${mutedCount} unavailable`,
  };
}

// evasionDisplayModel keeps the large-number defense display tolerant of bad
// fixture data without pushing normalization into the JSX tree.
export function evasionDisplayModel(evasion?: number): EvasionDisplayModel | undefined {
  if (evasion === undefined) {
    return undefined;
  }

  return {
    value: clampCount(evasion),
  };
}

function shieldState(index: number, filledCount: number, spentCount: number): ArmorShieldState {
  if (index < filledCount) {
    return "filled";
  }
  if (index < filledCount + spentCount) {
    return "spent";
  }
  return "muted";
}

function clampCount(value: number, max = Number.MAX_SAFE_INTEGER): number {
  if (!Number.isFinite(value)) {
    return 0;
  }

  return Math.min(Math.max(Math.trunc(value), 0), max);
}
