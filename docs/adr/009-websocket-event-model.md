# ADR-009: WebSocket notifications carry identity and version, not state

- **Status:** Accepted
- **Date:** 2026-10-02
- **Deciders:** @Tony5897

## Context

Pushing full entity state over sockets creates ordering and staleness problems and duplicates authorization logic.

## Decision

Realtime events carry `{type, entityType, entityId, version, occurredAt, traceId}` only. Clients invalidate TanStack Query caches and refetch over authorized HTTP. On every (re)connect, clients invalidate all active org queries, so convergence never depends on receiving every event. Each subscription is authorized per topic. Each connection has a bounded send queue (slow consumers are disconnected), heartbeats, size limits and an Origin check.

## Alternatives considered

| Option | Why not chosen |
|---|---|
| State-carrying events | Out-of-order and authorization complexity |
| Server-Sent Events | Viable, but the presence and subscribe protocol benefits from bidirectional messages |

## Consequences

- **Positive:** HTTP stays the single authoritative read path; reconnects are simple.
- **Negative:** each event costs an extra fetch.
- **Follow-ups:** convergence E2E test; slow-consumer test.

## Revisit if

Refetch amplification becomes a measured load problem.
