import { defineConfig } from "@hey-api/openapi-ts";

// Generates TypeScript types and Zod schemas from the contract (ADR-011).
// Requests go through the client's own apiFetch (CSRF, idempotency, problem
// mapping), so no HTTP client is generated.
export default defineConfig({
  input: "../../api/openapi.yaml",
  output: { path: "../../packages/api-client/src/gen", format: false, lint: false },
  plugins: ["@hey-api/typescript", { name: "zod", metadata: true }],
});
