# ADR-011: OpenAPI contract version

- **Status:** Proposed
- **Date:** 2026-10-02
- **Deciders:** @Tony5897

## Context

The API contract (`api/openapi.yaml`) generates the Go server interfaces and the TypeScript client. OpenAPI 3.2.1 is the newest specification, but generator support may lag behind it.

## Decision

**Pending a timeboxed spike (½ day, Gate 0).** Author a representative sample (problem+json, `If-Match`/`ETag`, `Idempotency-Key`, cursor pagination, a `oneOf` event) and run it through `oapi-codegen` (strict server, std-http) and `@hey-api/openapi-ts`. Try 3.2.1, then 3.1.x, then 3.0.4. Choose the highest version that round-trips without hand patches. Record the results in this ADR.

## Alternatives considered

| Option | Why not chosen |
|---|---|
| Always newest | Risks broken generation |
| Code-first spec generation | The contract would follow the code instead of leading it |

## Consequences

- **Follow-ups:** the spike outcome, the chosen version, and a CI check that bundled-spec regeneration is clean.

## Revisit if

Either generator gains full support for a newer version (Renovate will surface it).
