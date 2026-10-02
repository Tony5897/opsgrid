# ADR-016: Frontend stack

- **Status:** Accepted
- **Date:** 2026-10-02
- **Deciders:** @Tony5897

## Context

The SPA needs type-safe URL state (filters, dates, selection), server-state caching driven by realtime invalidation, accessible primitives, an accessible drag-and-drop dispatch board, and fast tooling.

## Decision

React 19 with React Compiler · Vite 8 · TypeScript (strict) · **TanStack Router** (file-based, Zod-validated search params) · **TanStack Query** (all server state) · TanStack Table + Virtual · React Hook Form + Zod · **Tailwind CSS v4** with OKLCH design tokens · **shadcn/ui** (Radix primitives, owned code) · dnd-kit (with a keyboard alternative) · cmdk · sonner · Recharts · Temporal API (polyfilled where not native) · **Biome** (lint + format) · Vitest browser mode + Testing Library · MSW · Storybook · Playwright. pnpm workspaces; the generated client lives in `packages/api-client`.

## Alternatives considered

| Option | Why not chosen |
|---|---|
| Next.js / RSC | SSR is not needed for an authenticated console; the BFF already exists in Go |
| React Router | Weaker typed search-param story |
| Material UI / Chakra | Harder to build a distinctive, token-driven design; heavier runtime |
| ESLint + Prettier | Two slower tools where one suffices |

## Consequences

- **Positive:** URL-as-state, typed end to end, owned component code.
- **Negative:** more libraries to keep current (Renovate).

## Revisit if

React Compiler or TanStack Router causes blocking defects without a workaround.
