# OpsGrid

A multi-tenant, real-time field-operations platform, built to explore relational isolation, transactional correctness, asynchronous delivery, realtime synchronization, observability and production failure handling.

> **Status:** Gate 0 (Foundation), in active development. Each capability below is listed with the gate that will prove it. Claims are linked to evidence only once proven; see the [evidence ledger](docs/evidence/ledger.md).

## What it does

Dispatchers drag work orders onto technicians' schedules on a live board. The database proves the technician is free and the change commits atomically with its audit record and its outbound event. Every other connected session updates without a refresh, and background workers handle notifications and reports. Each organization's data is isolated at three layers: application, relational constraints, and PostgreSQL Row-Level Security.

## Engineering highlights (planned → proven)

| Capability | Mechanism | Gate |
|---|---|---|
| Tenant isolation | App authorization + composite tenant FKs + forced RLS | G1 |
| No double-booking | PostgreSQL exclusion constraint on `tstzrange` | G3 |
| Safe concurrent edits | Versioned aggregates, `ETag` / `If-Match`, 412 + diff UI | G4 |
| Safe retries | `Idempotency-Key` claimed inside the business transaction | G4 |
| Live updates | Authorized WebSocket topics, bounded queues, reconnect convergence | G5 |
| Reliable events | Transactional outbox → Redis Streams, idempotent consumers, DLQ | G6 |
| Direct uploads | Tenant-authorized presigned object-storage URLs | G7 |
| End-to-end tracing | OpenTelemetry across HTTP, SQL, outbox and workers | G8 |
| Verified security | Threat model + OWASP ASVS 5.0.0 mapping | G9 |
| Measured performance | k6 scenarios, failure injection, published results | G10 |

## Stack

Go 1.27 · PostgreSQL 18 · Redis 8 · Keycloak (OIDC, BFF sessions) · React 19 + TypeScript · TanStack Router/Query · Tailwind CSS v4 · OpenTelemetry · Prometheus · Tempo · Loki · Grafana · Docker Compose · GitHub Actions

## Local setup

Requirements: Docker (or a compatible runtime), Go 1.27+, Node 24 LTS, pnpm. Run `make doctor` to check your environment.

```bash
git clone <repo-url> opsgrid && cd opsgrid
make dev
```

## Documentation

- [Implementation plan](IMPLEMENTATION_PLAN.md)
- [Architecture overview](docs/architecture/overview.md)
- [Architecture decision records](docs/adr/README.md)
- [Evidence ledger](docs/evidence/ledger.md)
- [Contributing](docs/CONTRIBUTING.md)

## Tradeoffs

OpsGrid is intentionally a modular monolith, runs on Docker Compose rather than Kubernetes, and uses Redis Streams rather than Kafka. Each choice and its "revisit if" conditions are recorded in the [ADRs](docs/adr/README.md).
