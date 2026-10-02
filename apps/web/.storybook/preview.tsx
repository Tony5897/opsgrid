import type { Decorator, Preview } from "@storybook/react-vite";
import { useEffect } from "react";
import { TooltipProvider } from "../src/components/ui/tooltip";
import "../src/index.css";

/** Mirrors the app: theme and density are attributes on <html>. */
const withAppearance: Decorator = (Story, context) => {
  const { theme, density } = context.globals as { theme: string; density: string };
  useEffect(() => {
    const root = document.documentElement;
    if (theme === "system") delete root.dataset.theme;
    else root.dataset.theme = theme;
    if (density === "compact") root.dataset.density = "compact";
    else delete root.dataset.density;
  }, [theme, density]);
  return (
    <TooltipProvider delayDuration={200}>
      <div className="bg-canvas p-6 text-text-primary">
        <Story />
      </div>
    </TooltipProvider>
  );
};

const preview: Preview = {
  decorators: [withAppearance],
  globalTypes: {
    theme: {
      description: "Color theme",
      toolbar: {
        title: "Theme",
        icon: "mirror",
        items: [
          { value: "light", title: "Light" },
          { value: "dark", title: "Dark" },
          { value: "system", title: "System" },
        ],
        dynamicTitle: true,
      },
    },
    density: {
      description: "Density",
      toolbar: {
        title: "Density",
        icon: "component",
        items: [
          { value: "comfortable", title: "Comfortable" },
          { value: "compact", title: "Compact" },
        ],
        dynamicTitle: true,
      },
    },
  },
  initialGlobals: { theme: "light", density: "comfortable" },
  parameters: {
    layout: "fullscreen",
    controls: { expanded: true },
    a11y: {
      // Violations fail the story tests, not just warn.
      test: "error",
      options: { runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"] } },
    },
  },
};

export default preview;
