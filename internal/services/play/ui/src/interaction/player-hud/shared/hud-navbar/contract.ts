import type { HUDNavbarTab } from "../contract";

export type HUDNavbarProps = {
  activeTab: HUDNavbarTab;
  isSidebarOpen: boolean;
  onSidebarOpenChange: (open: boolean) => void;
  onTabChange: (tab: HUDNavbarTab) => void;
  tabsWithUpdates?: Map<HUDNavbarTab, number>;
};
