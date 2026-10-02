# ADR-007: Transactional outbox for event publication

- **Status:** Accepted
- **Date:** 2026-10-02
- **Deciders:** @Tony5897

## Context

Updating the database and then publishing to a broker is a dual write: a crash in between loses the event or publishes a phantom one.

## Decision

Every command writes its business change, audit event and `outbox_events` row in **one transaction**. A separate `relay` process claims unpublished rows with `FOR UPDATE SKIP LOCKED`, publishes them, and marks `published_at`. It wakes on `LISTEN/NOTIFY` with a 1s polling fallback. Re-publication after a crash is expected; consumers are idempotent. Events carry the producer's W3C `traceparent`.

## Alternatives considered

| Option | Why not chosen |
|---|---|
| Publish after commit | Loses events on crash |
| CDC via logical replication (Debezium) | Heavy infrastructure for one producer |

## Consequences

- **Positive:** event intent is never lost or invented; publication can lag but not diverge.
- **Negative:** at-least-once publication; extra table and process.
- **Follow-ups:** crash-point tests A–D; `outbox_oldest_event_seconds` metric and alert.

## Revisit if

Event volume makes polling the outbox table a measured bottleneck.
