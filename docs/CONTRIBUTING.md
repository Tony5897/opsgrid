# Contributing to OpsGrid

These conventions keep the system's guarantees intact. Read [`IMPLEMENTATION_PLAN.md`](../IMPLEMENTATION_PLAN.md) and the [architecture overview](architecture/overview.md) first.

## Getting started

```bash
make doctor     # verify toolchain, Docker and ports
make dev        # boot the full stack (Compose) + web dev server
make test       # unit tests (Go + web)
```

Run `make help` to list all targets.

## Architecture rules (enforced in CI)

1. **Layering.** `internal/<domain>/domain.go` is pure (stdlib + `platform/clock` only). `service.go` holds commands and queries. `repo_pg.go` holds SQL. `http.go` is the transport adapter. Domains never import `net/http` outside `http.go`, and never import another domain's repository.
2. **The command spine.** Every mutation runs inside `database.InTenantTx` and follows: idempotency claim → load (locked or version-predicated) → authorize against the loaded object → pure domain transition → versioned write → audit record → outbox append → idempotency completion. Handlers contain no business logic.
3. **Tenant context.** Only `platform/database` may begin a transaction. Never call `pool.Begin` or `pool.Query` on tenant tables directly.
4. **Authorization.** Call `authz.Can(ctx, principal, permission, resource)`. Never compare role names in business code. Cross-tenant access returns 404, not 403.
5. **Errors.** Return typed domain errors. `httpx.WriteError` maps them to RFC 9457 problem details. Database error text never reaches clients.
6. **SQL.** Queries live in `db/queries/*.sql` and are generated with sqlc. User input never becomes an identifier: sorting uses allowlisted enums mapped to prebuilt queries.
7. **Migrations.** Forward-only SQL files under `db/migrations/`, linted by squawk. Each `-- +goose Up` section starts with `SET LOCAL lock_timeout` and `SET LOCAL statement_timeout`, so a migration fails fast instead of queueing behind (and blocking) live traffic. Wrap dollar-quoted bodies in `-- +goose StatementBegin/End`. After Gate 4, schema changes follow expand-and-contract.
8. **Contract first.** Change `api/openapi.yaml`, run `make generate`, then implement. CI fails on a stale generated client or an unflagged breaking change.

## Never do this to make a test pass

- Disable RLS, or grant the runtime role table ownership or `BYPASSRLS`
- Drop or weaken a constraint
- Skip `If-Match` or `Idempotency-Key` handling
- Swallow an error, or add a lint suppression without a justification comment

Each of these is detected in CI by the schema test, the contract, or lint.

## Frontend rules

- Server state lives only in TanStack Query; URL state lives in router search params.
- Every component ships with Storybook stories covering its states (loading, empty, error, populated) in light and dark.
- Status is always shown as color + icon + text. Every interactive flow is keyboard-operable.
- No `dangerouslySetInnerHTML`. No tokens or secrets in client code.

## Commits and pull requests

- [Conventional Commits](https://www.conventionalcommits.org/), scoped by module: `feat(scheduling): reject overlapping assignments`.
- Small PRs (aim for under 400 changed lines). Squash-merge.
- Complete the PR template's evidence checklist. Update [`docs/evidence/ledger.md`](evidence/ledger.md) when a guarantee gains proof.
- Record significant decisions as an ADR in [`docs/adr/`](adr/) using the [template](adr/000-template.md).
