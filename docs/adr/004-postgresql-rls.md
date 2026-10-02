# ADR-004: PostgreSQL Row-Level Security as the last line of tenant isolation

- **Status:** Accepted
- **Date:** 2026-10-02
- **Deciders:** @Tony5897

## Context

Application authorization can have bugs, and a forgotten `WHERE organization_id = …` must not leak data. PostgreSQL row security is default-deny once enabled, but table owners and `BYPASSRLS` roles skip it, and connection pooling makes session-level settings dangerous.

## Decision

- `ENABLE` and `FORCE ROW LEVEL SECURITY` on every tenant table, with one simple policy each: `organization_id = (SELECT app.current_org_id())`, for both `USING` and `WITH CHECK`.
- `app.current_org_id()` returns `NULLIF(current_setting('app.organization_id', true), '')::uuid`. The `NULLIF` is required because a tx-local setting reads back as `''` (not NULL) on a reused pooled connection. The `(SELECT …)` wrapper lets the planner evaluate it once per statement.
- Tenant context is set **only** via `set_config(…, true)` (transaction-local) inside `platform/database.InTenantTx`. No other code path may begin a transaction on tenant tables (enforced by lint).
- The runtime role `opsgrid_app` does not own tables and has no `BYPASSRLS`. Migrations run as `opsgrid_migrator`.
- Policies stay tenant-ID-only, with no cross-table subqueries.

## Alternatives considered

| Option | Why not chosen |
|---|---|
| Application filtering only | A single forgotten predicate leaks data |
| Session-level SET on pooled connections | Context leaks between requests when a connection is reused |

## Consequences

- **Positive:** even deliberately careless SQL cannot cross tenants; this is demonstrable in tests.
- **Negative:** every tenant query needs a transaction; RLS adds planner considerations.
- **Follow-ups:** schema test (RLS enabled+forced, policy present, runtime role not owner/BYPASSRLS); a fail-closed test when context is missing.

## Revisit if

Measured RLS overhead violates a performance budget after index and policy tuning.
