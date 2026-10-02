import type { StorybookConfig } from "@storybook/react-vite";

const config: StorybookConfig = {
  framework: "@storybook/react-vite",
  stories: ["../src/**/*.stories.@(ts|tsx)", "../src/**/*.mdx"],
  addons: ["@storybook/addon-docs", "@storybook/addon-a11y", "@storybook/addon-vitest"],
  staticDirs: ["../public"],
  typescript: {
    // react-docgen needs no TypeScript compiler API (TypeScript 7 is native).
    reactDocgen: "react-docgen",
  },
  core: { disableTelemetry: true },
};

export default config;
