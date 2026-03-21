import type { Meta, StoryObj } from "@storybook/react-vite";
import { HUDNavbar } from "./HUDNavbar";
import { hudNavbarFixtures } from "./fixtures";

const meta = {
  title: "Interaction/Player HUD/Navbar",
  component: HUDNavbar,
  tags: ["autodocs"],
  args: {
    onTabChange: () => {},
  },
  parameters: {
    docs: {
      description: {
        component: "Top-level navigation bar for the player HUD with three tab surfaces.",
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
    tabsWithUpdates: new Map([["side-chat", 2]]),
  },
};
