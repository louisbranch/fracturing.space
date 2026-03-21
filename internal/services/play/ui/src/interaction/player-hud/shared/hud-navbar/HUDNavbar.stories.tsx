import type { Meta, StoryObj } from "@storybook/react-vite";
import { HUDNavbar } from "./HUDNavbar";
import { hudNavbarFixtures } from "./fixtures";

const meta = {
  title: "Interaction/Player HUD/Shared/Navbar",
  component: HUDNavbar,
  tags: ["autodocs"],
  args: {
    connectionState: "connected",
    isSidebarOpen: false,
    onSidebarOpenChange: () => {},
    onTabChange: () => {},
  },
  parameters: {
    docs: {
      description: {
        component: "Top-level navigation bar for the player HUD with a left drawer trigger and three tab surfaces.",
      },
    },
  },
} satisfies Meta<typeof HUDNavbar>;

export default meta;

type Story = StoryObj<typeof meta>;

export const OnStage: Story = {
  args: { activeTab: hudNavbarFixtures.onStage.activeTab },
};

export const Backstage: Story = {
  args: { activeTab: hudNavbarFixtures.backstage.activeTab },
};

export const SideChat: Story = {
  args: { activeTab: hudNavbarFixtures.sideChat.activeTab },
};

export const WithUpdates: Story = {
  args: {
    activeTab: hudNavbarFixtures.onStage.activeTab,
    connectionState: "connected",
    tabsWithUpdates: new Map([["side-chat", 2]]),
  },
};

export const Reconnecting: Story = {
  args: {
    activeTab: hudNavbarFixtures.onStage.activeTab,
    connectionState: "reconnecting",
  },
};

export const Disconnected: Story = {
  args: {
    activeTab: hudNavbarFixtures.onStage.activeTab,
    connectionState: "disconnected",
  },
};
