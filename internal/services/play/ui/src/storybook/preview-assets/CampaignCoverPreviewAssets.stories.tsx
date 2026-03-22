import type { Meta, StoryObj } from "@storybook/react-vite";
import { PreviewAssetGallery } from "./PreviewAssetGallery";
import { campaignCoverPreviewAssets } from "./fixtures";

function CampaignCoverPreviewAssetsGallery() {
  return (
    <PreviewAssetGallery
      assets={campaignCoverPreviewAssets}
      pageTitle="Campaign Cover Preview Assets"
      pageDescription="Stable campaign-cover imagery sourced from the checked-in asset catalogs for play-service mockups."
      sectionTitle="Campaign Covers"
      sectionDescription="Uses the full published campaign-cover catalog for future cover-bearing mockups."
      imageClassName="aspect-[3/2] w-full object-cover"
    />
  );
}

const meta = {
  title: "Reference/Preview Assets/Campaign Covers",
  component: CampaignCoverPreviewAssetsGallery,
  tags: ["autodocs"],
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Catalog view for the shared Storybook campaign-cover assets used by play-service mockups.",
      },
    },
  },
} satisfies Meta<typeof CampaignCoverPreviewAssetsGallery>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Catalog: Story = {};
