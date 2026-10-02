# ADR-008: Redis Streams for background work

- **Status:** Accepted
- **Date:** 2026-10-02
- **Deciders:** @Tony5897

## Context

Background jobs need durable queuing, consumer groups, redelivery of unacknowledged work, retries with backoff, and a dead-letter path. Redis is already present for presence, rate limits and fan-out.

## Decision

Use Redis Streams with one consumer group per handler. Dedupe uses a `processed_messages` row written in the **same PostgreSQL transaction** as the business effect. Delayed retries use a sorted set (`retry:{group}`) moved back to the stream by a scheduler (atomic Lua). Stuck messages are reclaimed with `XAUTOCLAIM`. After the maximum attempts, messages go to `stream:dlq`, which is visible and re-drivable in the UI. An `EventBus` interface keeps a broker swap possible.

## Alternatives considered

| Option | Why not chosen |
|---|---|
| Kafka | Operational weight unjustified at this scale |
| PostgreSQL-only queue (SKIP LOCKED / River) | Viable, but the project wants to demonstrate broker semantics and redelivery explicitly |
| NATS JetStream | A good fit, but adds a component Redis already covers |

## Consequences

- **Positive:** one infrastructure component serves several roles; explicit, testable delivery semantics.
- **Negative:** retry delay is hand-built; Redis persistence must be configured (AOF).
- **Follow-ups:** failure tests for redelivery and DLQ.

## Revisit if

Throughput, retention or replay requirements exceed what Redis Streams provides comfortably.
