# ADR-005: SQL-first data access with pgx and sqlc

- **Status:** Accepted
- **Date:** 2026-10-02
- **Deciders:** @Tony5897

## Context

The relational model, its constraints and its queries are core evidence for this project. ORMs hide SQL, encourage N+1 patterns and make RLS and locking semantics opaque.

## Decision

Write SQL by hand in `db/queries/*.sql` and generate typed Go with **sqlc** (pgx/v5 driver). Use `pgxpool` for pooling. Migrations are hand-written SQL managed by **goose**, embedded in the binary, linted by **squawk**. Dynamic sorting maps an allowlisted enum to prebuilt queries; user input never becomes an identifier.

## Alternatives considered

| Option | Why not chosen |
|---|---|
| GORM / ent | Hides the SQL and locking behavior being demonstrated |
| Raw pgx only | Loses compile-time typing of query results |

## Consequences

- **Positive:** visible, reviewable SQL; compile-time types; EXPLAIN-able queries.
- **Negative:** more files; dynamic filters need careful query composition.
- **Follow-ups:** `sqlc.yaml`; CI verifies that generated code is up to date.

## Revisit if

Query composition needs become so dynamic that the number of prebuilt variants is unmaintainable.
