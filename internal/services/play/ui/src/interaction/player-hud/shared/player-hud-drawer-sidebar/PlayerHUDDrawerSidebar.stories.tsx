import type { Meta, StoryObj } from "@storybook/react-vite";
import { PlayerHUDDrawerSidebar } from "./PlayerHUDDrawerSidebar";
import { playerHUDFixtureCatalog } from "../fixtures";

const meta = {
  title: "Interaction/Player HUD/Shared/Drawer Sidebar",
  component: PlayerHUDDrawerSidebar,
  args: {
    onClose: () => {},
  },
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof PlayerHUDDrawerSidebar>;

export default meta;

type Story = StoryObj<typeof meta>;

export const CampaignRoster: Story = {
  args: {
    navigation: playerHUDFixtureCatalog.onStage.campaignNavigation,
  },
};
