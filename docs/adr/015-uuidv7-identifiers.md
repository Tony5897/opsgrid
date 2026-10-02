# ADR-015: UUIDv7 primary keys plus per-organization display numbers

- **Status:** Accepted
- **Date:** 2026-10-02
- **Deciders:** @Tony5897

## Context

Random UUIDv4 keys fragment B-tree indexes. Sequential integer IDs leak volume and enable enumeration. Humans need short references ("WO-1844").

## Decision

Primary keys are `uuid DEFAULT uuidv7()` (native in PostgreSQL 18): time-ordered and index-friendly. Work orders also get a per-organization `number`, allocated from `organization_counters` with `UPDATE … RETURNING` in the same transaction, unique on `(organization_id, number)`. The API exposes UUIDs; the UI displays numbers.

## Alternatives considered

| Option | Why not chosen |
|---|---|
| bigserial | Enumerable, and leaks global volume |
| UUIDv4 | Index locality |
| Prefixed TypeIDs | Extra codec work across SQL, OpenAPI and TS for marginal benefit |

## Consequences

- **Positive:** good locality; no enumeration; readable references.
- **Negative:** UUIDv7 exposes creation time (acceptable).

## Revisit if

Display numbers need to be globally unique across organizations.
