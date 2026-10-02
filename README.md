# OpsGrid

A multi-tenant, real-time field-operations platform, built to explore relational isolation, transactional correctness, asynchronous delivery, realtime synchronization, observability and production failure handling.

> **Status:** Gate 0 (Foundation) complete; Gate 1 (identity and tenancy) next. Each capability below is listed with the gate that will prove it. Claims are linked to evidence only once proven; see the [evidence ledger](docs/evidence/ledger.md).

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

Go 1.27 · PostgreSQL 18 · Redis 8 · Keycloak (OIDC, BFF sessions) · Garage/R2 (S3) · React 19 + TypeScript · TanStack Router/Query · Tailwind CSS v4 · OpenAPI 3.1 · OpenTelemetry · Prometheus · Tempo · Loki · Grafana · Docker Compose · GitHub Actions

## Local setup

Requirements: a Docker runtime (OrbStack, Docker Desktop or Colima), Go 1.27+, Node 24 LTS and pnpm 12. Run `make doctor` to check your machine.

```bash
git clone https://github.com/Tony5897/opsgrid.git && cd opsgrid
make setup   # toolchain, dependencies, browsers
make dev     # build and start the stack; waits until healthy
```

| Open | URL |
|---|---|
| OpsGrid | http://localhost:8080 |
| Hot-reload UI (`make web-dev`) | http://localhost:5173 |
| Component workbench (`make storybook`) | http://localhost:6006 |
| Grafana (`make dev-obs`) | http://localhost:3000 |
| Keycloak | http://localhost:8180 |
| Mailpit | http://localhost:8025 |

The full guide, covering every service, credential, command, test layer and troubleshooting step, is [docs/development.md](docs/development.md).

## Documentation

- [Developer guide](docs/development.md)
- [Implementation plan](IMPLEMENTATION_PLAN.md)
- [Architecture overview](docs/architecture/overview.md)
- [Architecture decision records](docs/adr/README.md)
- [Evidence ledger](docs/evidence/ledger.md)
- [Contributing](docs/CONTRIBUTING.md)

## Tradeoffs

OpsGrid is intentionally a modular monolith, runs on Docker Compose rather than Kubernetes, and uses Redis Streams rather than Kafka. Each choice and its "revisit if" conditions are recorded in the [ADRs](docs/adr/README.md).
