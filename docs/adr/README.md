# Architecture Decision Records

Format: MADR-style ([template](000-template.md)). Number decisions sequentially and never renumber them. Supersede old decisions instead of editing their outcome.

| ADR | Title | Status |
|---|---|---|
| [001](001-modular-monolith.md) | Modular monolith with separate API, relay and worker processes | Accepted |
| [002](002-go-backend.md) | Go for the backend | Accepted |
| [003](003-shared-schema-multitenancy.md) | Shared-schema multi-tenancy with tenant-aware composite keys | Accepted |
| [004](004-postgresql-rls.md) | PostgreSQL Row-Level Security as the last line of tenant isolation | Accepted |
| [005](005-sql-first-data-access.md) | SQL-first data access with pgx and sqlc | Accepted |
| [006](006-oidc-bff.md) | OpenID Connect via a backend-for-frontend session | Accepted |
| [007](007-transactional-outbox.md) | Transactional outbox for event publication | Accepted |
| [008](008-redis-streams.md) | Redis Streams for background work | Accepted |
| [009](009-websocket-event-model.md) | WebSocket notifications carry identity and version, not state | Accepted |
| [010](010-no-kubernetes-initially.md) | Docker Compose, not Kubernetes, as the canonical runtime | Accepted |
| [011](011-openapi-version.md) | OpenAPI contract version | Accepted |
| [012](012-problem-details-errors.md) | RFC 9457 Problem Details for all API errors | Accepted |
| [013](013-sessions-in-postgresql.md) | Server-side sessions stored in PostgreSQL | Accepted |
| [014](014-stdlib-router.md) | Go standard library ServeMux for routing | Accepted |
| [015](015-uuidv7-identifiers.md) | UUIDv7 primary keys plus per-organization display numbers | Accepted |
| [016](016-frontend-stack.md) | Frontend stack | Accepted |
| [017](017-local-object-storage.md) | Local S3-compatible object storage server | Proposed |
| [018](018-log-pipeline.md) | Logs: slog JSON on stdout; shipping decided by OTel log maturity | Accepted |
| [019](019-csrf-defense.md) | CSRF defense in depth | Accepted |
| [020](020-spa-served-by-bff.md) | The Go API serves the built SPA (same origin) | Accepted |
| [021](021-cloudevents-envelope.md) | CloudEvents 1.0 envelope for internal events | Accepted |
