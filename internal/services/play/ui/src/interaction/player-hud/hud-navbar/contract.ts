import type { HUDNavbarTab } from "../shared/contract";

export type HUDNavbarProps = {
  activeTab: HUDNavbarTab;
  onTabChange: (tab: HUDNavbarTab) => void;
  tabsWithUpdates?: Map<HUDNavbarTab, number>;
};
