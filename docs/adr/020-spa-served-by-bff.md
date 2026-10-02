# ADR-020: The Go API serves the built SPA (same origin)

- **Status:** Accepted
- **Date:** 2026-10-02
- **Deciders:** @Tony5897

## Context

Cross-origin SPA/API deployments need CORS with credentials and complicate cookies and CSP.

## Decision

In production, the built SPA is embedded in the API binary (`embed.FS`) and served with immutable caching for hashed assets, `no-cache` for `index.html`, and an SPA fallback for client routes. In development, Vite runs on :5173 and proxies `/v1`, `/auth` and `/v1/realtime` (ws) to the API on :8080.

## Alternatives considered

| Option | Why not chosen |
|---|---|
| Separate static host or CDN | Cross-origin cookies and CORS |
| Nginx sidecar | Extra component |

## Consequences

- **Positive:** no CORS; strict CSP `script-src 'self'`; one deployable.
- **Negative:** a frontend change rebuilds the API image (acceptable).

## Revisit if

Global edge delivery of static assets becomes a measured need.
