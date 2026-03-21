import type { Preview } from "@storybook/react-vite";
import { createElement } from "react";
import { themes } from "storybook/theming";
import "../src/styles.css";

const preview: Preview = {
  decorators: [
    (Story, context) => {
      if (context.title.indexOf("Interaction/Player HUD/") !== 0) {
        return createElement(Story);
      }

      return createElement(
        "div",
        { className: "play-density-hud h-full w-full" },
        createElement(Story),
      );
    },
  ],
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
