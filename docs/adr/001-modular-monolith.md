# ADR-001: Modular monolith with separate API, relay and worker processes

- **Status:** Accepted
- **Date:** 2026-10-02
- **Deciders:** @Tony5897

## Context

OpsGrid has a single engineering owner and roughly fourteen domain modules. Many important invariants cross modules: an assignment references a work order and a technician membership, and audit and outbox rows must commit atomically with business changes. Microservices would turn those invariants into distributed transactions and sagas before there is any scaling need.

## Decision

Build one Go module with strict internal domain packages (`internal/<domain>`), deployed as three processes from the same codebase: `api` (HTTP, WebSocket, serves the SPA), `relay` (outbox publisher) and `worker` (stream consumers). Domains talk to each other through exported service interfaces, never through each other's tables or repositories. `depguard` enforces this in CI.

## Alternatives considered

| Option | Why not chosen |
|---|---|
| Microservices per domain | Distributed transactions for invariants that PostgreSQL enforces trivially; operational overhead with no measured benefit |
| Single process (API runs workers) | Couples request latency to background load; hides the async boundary the system must demonstrate |

## Consequences

- **Positive:** cross-domain invariants stay in one ACID transaction; one build, one migration stream; extraction paths remain credible thanks to the enforced boundaries.
- **Negative:** all modules share a release cadence and a database; a hot module cannot scale independently of the API process.
- **Follow-ups:** depguard rules; `internal/app` is the only composition root; an architecture doc shows a future extraction path.

## Revisit if

A module needs an independent scaling or availability profile that is measured, not hypothetical, or the team grows to the point where independent deployability outweighs transactional simplicity.
