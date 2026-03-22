import type { Meta, StoryObj } from "@storybook/react-vite";
import { PreviewAssetGallery } from "./PreviewAssetGallery";
import { portraitSheetPreviewAssets } from "./fixtures";

function PortraitPreviewAssetsGallery() {
  return (
    <PreviewAssetGallery
      assets={portraitSheetPreviewAssets}
      pageTitle="Portrait Preview Assets"
      pageDescription="Stable user-portrait sheets sourced from the checked-in asset catalogs for play-service mockups."
      sectionTitle="Portraits"
      sectionDescription="Uses the published people-avatar catalog and shows each asset as the full 4-up portrait mosaic at the manifest 2:3 ratio."
      imageClassName="aspect-[2/3] w-full object-contain"
    />
  );
}

const meta = {
  title: "Reference/Preview Assets/Portraits",
  component: PortraitPreviewAssetsGallery,
  tags: ["autodocs"],
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Catalog view for the shared Storybook portrait-sheet assets used by play-service mockups.",
      },
    },
  },
} satisfies Meta<typeof PortraitPreviewAssetsGallery>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Catalog: Story = {};
