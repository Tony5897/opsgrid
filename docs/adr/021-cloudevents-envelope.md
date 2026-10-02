# ADR-021: CloudEvents 1.0 envelope for internal events

- **Status:** Accepted
- **Date:** 2026-10-02
- **Deciders:** @Tony5897

## Context

Outbox rows, stream messages and realtime notifications need a stable, versioned envelope with trace linkage.

## Decision

Use CloudEvents 1.0 attribute names in JSON: `specversion`, `id` (UUIDv7, also the dedupe key), `source` (`opsgrid/<module>`), `type` (`com.opsgrid.work_order.completed.v1`), `subject` (entity ID), `time`, `datacontenttype`, `dataschema`, plus the extensions `organizationid` and `traceparent` (CloudEvents distributed-tracing extension). Breaking payload changes bump the `.vN` type suffix.

## Alternatives considered

| Option | Why not chosen |
|---|---|
| Ad-hoc envelope | Reinvents a standard |

## Consequences

- **Positive:** standard tooling and vocabulary; explicit versioning.

## Revisit if

Never; the envelope is extensible.
