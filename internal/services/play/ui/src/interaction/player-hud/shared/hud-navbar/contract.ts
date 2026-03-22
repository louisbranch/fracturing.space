import type { HUDConnectionState, HUDNavbarTab } from "../contract";

export type HUDNavbarProps = {
  activeTab: HUDNavbarTab;
  aiDebugEnabled?: boolean;
  connectionState: HUDConnectionState;
  isSidebarOpen: boolean;
  onSidebarOpenChange: (open: boolean) => void;
  onTabChange: (tab: HUDNavbarTab) => void;
  tabsWithUpdates?: Map<HUDNavbarTab, number>;
};
