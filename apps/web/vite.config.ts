import { fileURLToPath } from "node:url";
import babel from "@rolldown/plugin-babel";
import { storybookTest } from "@storybook/addon-vitest/vitest-plugin";
import tailwindcss from "@tailwindcss/vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import react, { reactCompilerPreset } from "@vitejs/plugin-react";
import { playwright } from "@vitest/browser-playwright";
import { defineConfig } from "vite";

const apiTarget = process.env.OPSGRID_API_URL ?? "http://localhost:8080";

export default defineConfig({
  plugins: [
    // Must run before the React plugin so generated route files are transformed.
    tanstackRouter({ target: "react", autoCodeSplitting: true }),
    react(),
    babel({ presets: [reactCompilerPreset()] }),
    tailwindcss(),
  ],
  resolve: {
    alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) },
  },
  server: {
    port: 5173,
    strictPort: true,
    // Same-origin in dev too (ADR-020): the browser only ever talks to :5173.
    proxy: {
      "/v1/realtime": { target: apiTarget.replace(/^http/, "ws"), ws: true },
      "/v1": { target: apiTarget, changeOrigin: false },
      "/auth": { target: apiTarget, changeOrigin: false },
    },
  },
  build: {
    target: "es2023",
    sourcemap: true,
    manifest: true, // read by scripts/check-bundle-budget.mjs
    // The real budget (<=180 KB gzip initial JS) is enforced by
    // scripts/check-bundle-budget.mjs; this raw-size warning would be noise.
    chunkSizeWarningLimit: 800,
  },
  test: {
    projects: [
      {
        extends: true,
        test: {
          name: "unit",
          include: ["src/**/*.test.{ts,tsx}"],
          setupFiles: ["./src/test/setup.ts"],
          browser: {
            enabled: true,
            headless: true,
            provider: playwright(),
            viewport: { width: 1280, height: 800 },
            instances: [{ browser: "chromium" }],
          },
        },
      },
      {
        extends: true,
        plugins: [
          storybookTest({ configDir: fileURLToPath(new URL("./.storybook", import.meta.url)) }),
        ],
        test: {
          name: "storybook",
          setupFiles: ["./.storybook/vitest.setup.ts"],
          browser: {
            enabled: true,
            headless: true,
            provider: playwright(),
            instances: [{ browser: "chromium" }],
          },
        },
      },
    ],
  },
});
