# ADR-003: Shared-schema multi-tenancy with tenant-aware composite keys

- **Status:** Accepted
- **Date:** 2026-10-02
- **Deciders:** @Tony5897

## Context

Many organizations share one deployment. Options range from database-per-tenant through schema-per-tenant to shared tables. Isolation must be provable, and the database itself should reject structurally cross-tenant references.

## Decision

Use shared tables. Every tenant-owned table has `organization_id uuid NOT NULL` and `UNIQUE (organization_id, id)`. Every reference between tenant-owned tables is a **composite foreign key** `(organization_id, <parent>_id) → parent (organization_id, id)`, which makes a cross-tenant reference structurally impossible. A schema test asserts the pattern for every tenant table.

## Alternatives considered

| Option | Why not chosen |
|---|---|
| Database per tenant | Operationally heavy; migrations fan out; cross-tenant platform metrics are hard |
| Schema per tenant | Migration fan-out and catalog bloat; connection-pool `search_path` hazards |

## Consequences

- **Positive:** a single migration stream; the database enforces isolation at the relationship level.
- **Negative:** every index and FK carries `organization_id`; noisy-neighbor effects need rate limits.
- **Follow-ups:** `test/integration/schema_test.go` checks composite keys and FKs for every tenant table.

## Revisit if

A tenant requires physical data residency or isolation guarantees that shared tables cannot satisfy.
