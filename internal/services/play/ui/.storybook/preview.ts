import type { Preview } from "@storybook/react-vite";
import { themes } from "storybook/theming";
import "../src/styles.css";

const preview: Preview = {
  parameters: {
    controls: {
      expanded: true,
    },
    docs: {
      theme: themes.dark,
    },
    layout: "padded",
  },
};

export default preview;
